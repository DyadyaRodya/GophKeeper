package app

import "os"

type App struct{}

func NewApp() (*App, error) {
	return &App{}, nil
}

func (app *App) RunServer() error {
	return nil
}

func (app *App) RunClient() error {
	return nil
}

func (app *App) ShutdownServer(os.Signal) error {
	return nil
}

func (app *App) ShutdownClient(os.Signal) error {
	return nil
}
