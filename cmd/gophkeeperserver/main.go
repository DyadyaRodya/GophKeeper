package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/DyadyaRodya/GophKeeper/internal/app/server"
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

	serverApp, err := server.NewApp()
	if err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		s := <-c
		err = serverApp.Shutdown(s)
		if err != nil {
			panic(err)
		}
	}()
	err = serverApp.Run()
	if err != nil {
		panic(err)
	}
}
