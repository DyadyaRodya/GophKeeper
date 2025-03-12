package client

import (
	"log"
	"os"

	"go.uber.org/zap"

	"github.com/DyadyaRodya/GophKeeper/internal/app/logger"
)

// App client app service
type App struct {
	appLogger *zap.Logger
}

// NewApp constructor for client App
func NewApp() (*App, error) {
	appLogger, err := logger.InitLogger("debug")
	if err != nil {
		//log.Printf("Config %+v\n", *appConfig)
		log.Fatalf("Cannot initialize logger %+v\n", err)
	}
	//localKeyStorage, err := local.NewKeyStorageLocal()
	//cs := domainservices.NewDataConvertor()
	return &App{
		appLogger: appLogger,
	}, nil
}

// Run starts client App service
func (app *App) Run() error {
	return nil
}

// Shutdown perform graceful shutdown for client App
func (app *App) Shutdown(os.Signal) error {
	return nil
}
