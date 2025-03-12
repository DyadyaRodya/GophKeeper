package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/DyadyaRodya/GophKeeper/internal/auth"
	"github.com/DyadyaRodya/GophKeeper/pkg/encryption"
	pb "github.com/DyadyaRodya/GophKeeper/proto/v1"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/DyadyaRodya/GophKeeper/internal/app/logger"
	domainservices "github.com/DyadyaRodya/GophKeeper/internal/domain/services"
	grpchandlers "github.com/DyadyaRodya/GophKeeper/internal/handlers/grpc"
	filestorage "github.com/DyadyaRodya/GophKeeper/internal/repositories/filestorage/server"
	pgxrepo "github.com/DyadyaRodya/GophKeeper/internal/repositories/postgres"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases"
)

// App server app service
type App struct {
	appConfig  *Config
	gh         *grpchandlers.Handlers
	grpcServer *grpc.Server
	appLogger  *zap.Logger
	close      func()
}

// NewApp constructor for server App
func NewApp(defaultGrpcAddress, defaultLogLevel, defaultStorageDir string) (*App, error) {
	ctx := context.Background()
	appConfig := InitConfigFromCMD(defaultGrpcAddress, defaultLogLevel, defaultStorageDir)
	appLogger, err := logger.InitLogger(appConfig.LogLevel)
	if err != nil {
		log.Printf("Config %+v\n", *appConfig)
		log.Fatalf("Cannot initialize logger %+v\n", err)
	}

	appLogger.Info("Config", zap.Any("config", appConfig))

	secretKeyString := os.Getenv("SECRET_KEY")
	var secretKey []byte
	if secretKeyString == "" {
		secretKey, err = newSecretKey(32)
		if err != nil {
			appLogger.Fatal("Error generating secret key", zap.Error(err))
		}

		base64Text := make([]byte, base64.URLEncoding.EncodedLen(len(secretKey)))
		base64.URLEncoding.Encode(base64Text, secretKey)
		appLogger.Debug("New secret key", zap.ByteString("SECRET_KEY", base64Text))
	} else {
		appLogger.Debug("old secret key", zap.String("SECRET_KEY", secretKeyString))

		secretKey = make([]byte, base64.URLEncoding.DecodedLen(len(secretKeyString))-1)
		var n int
		n, err = base64.URLEncoding.Decode(secretKey, []byte(secretKeyString))
		appLogger.Debug("after decoding secret key", zap.Int("n", n), zap.Error(err))
		if err != nil {
			appLogger.Fatal("Error decoding secret key", zap.Error(err))
		}
	}
	// services
	jwts := auth.NewJWTService(secretKey, appConfig.TokenTTL)
	ps := domainservices.NewPasswordDomainService(appConfig.PasswordComplexityConfig)
	us := domainservices.NewUsernameDomainService(appConfig.UsernameConfig)
	uuidg := domainservices.NewUUID4Generator()
	es := &encryption.EncryptService{}
	// repository
	pool, err := pgxpool.New(ctx, appConfig.DSN)
	if err != nil {
		appLogger.Fatal("Cannot create connection pool to database", zap.String("DATABASE_DSN", ""), zap.Error(err))
	}
	repo := pgxrepo.NewStorePGX(pool, appLogger)
	if err = repo.InitSchema(ctx); err != nil {
		appLogger.Fatal("Cannot initialize database", zap.Error(err))
	}
	// storage
	storage := filestorage.NewDataStorageServer(appConfig.StorageDir)
	// usecases
	serverRegisterUsecase := usecases.NewServerRegisterUsecase(repo, uuidg, ps, us, appConfig.SaltSize, es)
	serverLoginUsecase := usecases.NewServerLoginUsecase(repo, ps)
	serverRecoverUsecase := usecases.NewServerRecoverUsecase(repo, ps, appConfig.SaltSize, es)
	serverUpdatePasswordUsecase := usecases.NewServerUpdatePasswordUsecase(repo, ps, appConfig.SaltSize, es)
	serverDataSaveUsecase := usecases.NewServerSaveDataUsecase(repo, uuidg, storage)
	serverDataReadUsecase := usecases.NewServerReadDataUsecase(repo, storage)
	serverDataListUsecase := usecases.NewServerListDataUsecase(repo)

	// handlers
	gh := grpchandlers.NewHandlers(
		jwts,
		serverDataListUsecase,
		serverLoginUsecase,
		serverDataReadUsecase,
		serverRecoverUsecase,
		serverRegisterUsecase,
		serverDataSaveUsecase,
		serverUpdatePasswordUsecase,
	)

	closer := func() {
		pool.Close()
	}
	return &App{
		appConfig: appConfig,
		gh:        gh,
		appLogger: appLogger,
		close:     closer,
	}, nil
}

// Run starts server App service
func (a *App) Run() error {
	if a.appConfig.EnableHTTPS {
		ensureCertAndKeyExist(a.appLogger)
	}
	listen, err := net.Listen("tcp", a.appConfig.GrpcAddress)
	if err != nil {
		a.appLogger.Error("Starting gRPC error", zap.Error(err))
		return err
	}

	var s *grpc.Server
	if a.appConfig.EnableHTTPS {
		creds, _ := credentials.NewServerTLSFromFile("goshortener.cert.pem", "goshortener.key.pem")
		s = grpc.NewServer(grpc.Creds(creds), grpc.ChainUnaryInterceptor(logging.UnaryServerInterceptor(logger.InterceptorLogger(a.appLogger))))
	} else {
		s = grpc.NewServer(grpc.ChainUnaryInterceptor(logging.UnaryServerInterceptor(logger.InterceptorLogger(a.appLogger))))
	}

	pb.RegisterGophKeeperServiceServer(s, a.gh)
	a.grpcServer = s

	a.appLogger.Info("gRPC started at", zap.String("grpc_address", a.appConfig.GrpcAddress))
	if err := s.Serve(listen); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		a.appLogger.Error("Serving gRPC error", zap.Error(err))
		return err
	}
	return nil
}

// Shutdown perform graceful shutdown for server App
func (a *App) Shutdown(signal os.Signal) error {
	defer a.close()

	a.appLogger.Info("Stopped server on signal", zap.String("signal", signal.String()))
	a.grpcServer.GracefulStop()
	return nil
}

func newSecretKey(size int) ([]byte, error) {
	secretKey := make([]byte, size)
	_, err := rand.Read(secretKey)
	if err != nil {
		return nil, err
	}
	return secretKey, nil
}
