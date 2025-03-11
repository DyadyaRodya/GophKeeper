package server

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/DyadyaRodya/GophKeeper/internal/app/logger"
	domainservices "github.com/DyadyaRodya/GophKeeper/internal/domain/services"
	pgxrepo "github.com/DyadyaRodya/GophKeeper/internal/repositories/postgres"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases"
)

// App server app service
type App struct{}

// NewApp constructor for server App
func NewApp() (*App, error) {
	appLogger, err := logger.InitLogger("debug")
	if err != nil {
		//log.Printf("Config %+v\n", *appConfig)
		log.Fatalf("Cannot initialize logger %+v\n", err)
	}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	// repository
	pool, err := pgxpool.New(ctx, "")
	if err != nil {
		appLogger.Fatal("Cannot create connection pool to database", zap.String("DATABASE_URI", ""), zap.Error(err))
	}
	// domain services
	passwordService := domainservices.NewPasswordDomainService(&domainservices.PasswordComplexityConfig{
		Length:          8,
		NumberOfDigits:  1,
		NumberOfUpper:   1,
		NumberOfLower:   1,
		NumberOfSpecial: 1,
	})
	//closeStorage := pool.Close
	repo := pgxrepo.NewStorePGX(pool, appLogger)

	// usecases
	u := usecases.NewServerLoginUsecase(repo, passwordService)
	u.Handle(ctx, "", "")
	return &App{}, nil
}

// Run starts server App service
func (app *App) Run() error {
	return nil
}

// Shutdown perform graceful shutdown for server App
func (app *App) Shutdown(os.Signal) error {
	return nil
}
