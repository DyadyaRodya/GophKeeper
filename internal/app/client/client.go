package client

import (
	"context"
	"log"

	"google.golang.org/grpc/credentials/insecure"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	domainservices "github.com/DyadyaRodya/GophKeeper/internal/domain/services"
	"github.com/DyadyaRodya/GophKeeper/internal/handlers/cli"
	"github.com/DyadyaRodya/GophKeeper/internal/repositories/filestorage/local"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases"
	"github.com/DyadyaRodya/GophKeeper/pkg/encryption"
	pb "github.com/DyadyaRodya/GophKeeper/proto/v1"

	"github.com/DyadyaRodya/GophKeeper/internal/app/logger"
	gateway "github.com/DyadyaRodya/GophKeeper/internal/gateways/grpc"
)

// App client app service
type App struct {
	appLogger *zap.Logger
	appConfig *Config
	h         *cli.Handlers
	close     func()
}

// NewApp constructor for client App
func NewApp(defaultGrpcAddress string, defaultDebug bool) (*App, error) {
	appConfig := InitConfigFromCMD(defaultGrpcAddress, defaultDebug)
	appLogger, err := logger.InitLogger(appConfig.Loglevel)
	if err != nil {
		log.Printf("Config %+v\n", *appConfig)
		log.Fatalf("Cannot initialize logger %+v\n", err)
	}
	appLogger.Info("Server info", zap.String("GRPC_ADDRESS", appConfig.GrpcAddress))

	// services
	ps := domainservices.NewPasswordDomainService(appConfig.PasswordComplexityConfig)
	us := domainservices.NewUsernameDomainService(appConfig.UsernameConfig)
	uuidg := domainservices.NewUUID4Generator()
	es := &encryption.EncryptService{}
	cs := domainservices.NewDataConvertor()

	// storages
	ds, err := local.NewDataStorageLocal()
	if err != nil {
		appLogger.Fatal("Error on NewDataStorageLocal", zap.Error(err))
	}
	lks, err := local.NewKeyStorageLocal()
	if err != nil {
		appLogger.Fatal("Error on NewKeyStorageLocal", zap.Error(err))
	}

	// gateway
	conn, err := grpc.NewClient(appConfig.GrpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		appLogger.Fatal("Error on grpc.NewClient()", zap.Error(err))
	}
	client := pb.NewGophKeeperServiceClient(conn)
	gw := gateway.NewClientGRPC(client)

	// usecases
	clientRegisterUsecase := usecases.NewClientRegisterUsecase(ps, us, es, lks, gw)
	clientLoginUsecase := usecases.NewClientLoginUsecase(es, lks, gw)
	clientRecoverUsecase := usecases.NewClientRecoverUsecase(ps, es, lks, gw)
	clientUpdatePasswordUsecase := usecases.NewClientUpdatePasswordUsecase(ps, es, lks, gw)
	clientDataSaveUsecase := usecases.NewClientSaveDataUsecase(cs, es, uuidg, ds, gw)
	clientDataReadUsecase := usecases.NewClientReadDataUsecase(cs, es, ds, gw)
	clientDataListUsecase := usecases.NewClientListDataUsecase(ds, gw)
	clientDataDeleteUsecase := usecases.NewClientDeleteDataUsecase(ds, gw)
	clientDataSyncUsecase := usecases.NewClientSyncDataUsecase(ds, gw)

	// handlers
	h := cli.NewHandlers(
		clientDataListUsecase,
		clientLoginUsecase,
		clientDataReadUsecase,
		clientRecoverUsecase,
		clientRegisterUsecase,
		clientDataSaveUsecase,
		clientDataDeleteUsecase,
		clientUpdatePasswordUsecase,
		clientDataSyncUsecase,
		appLogger,
		appConfig.Debug,
	)

	closer := func() {
		conn.Close()
	}
	return &App{
		appLogger: appLogger,
		appConfig: appConfig,
		h:         h,
		close:     closer,
	}, nil
}

// Run starts client App service
func (a *App) Run(ctx context.Context) error {
	a.appLogger.Info("Starting app")
	code, err := a.h.EnterMenu(ctx)
	a.appLogger.Info("App stopped with code", zap.Int("code", int(code)))
	if code == cli.HandlersProcessError {
		a.appLogger.Error("Error in app occurred", zap.Error(err))
		return err
	}
	return nil
}

// Close releases resources of client App
func (a *App) Close() error {
	a.close()
	a.appLogger.Info("Closed client app")
	return nil
}
