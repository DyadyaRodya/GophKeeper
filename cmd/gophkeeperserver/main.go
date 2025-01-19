package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/DyadyaRodya/GophKeeper/internal/app"
)

var buildVersion = "N/A" //nolint: gochecknoglobals // This var could be global
var buildDate = "N/A"    //nolint: gochecknoglobals // This var could be global
var buildCommit = "N/A"  //nolint: gochecknoglobals // This var could be global

func main() {
	fmt.Printf(
		"Build version: %s\nBuild date: %s\nBuild commit: %s\n",
		buildVersion,
		buildDate,
		buildCommit,
	)

	server, err := app.NewApp()
	if err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		s := <-c
		err = server.ShutdownServer(s)
		if err != nil {
			panic(err)
		}
	}()
	err = server.RunServer()
	if err != nil {
		panic(err)
	}
}
