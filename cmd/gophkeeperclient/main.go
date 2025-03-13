package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/DyadyaRodya/GophKeeper/internal/app/client"
)

var buildVersion = "N/A" //nolint: gochecknoglobals // This var could be global
var buildDate = "N/A"    //nolint: gochecknoglobals // This var could be global
var buildCommit = "N/A"  //nolint: gochecknoglobals // This var could be global

const (
	defaultGrpcAddress = `localhost:50051`
	defaultDebug       = false
)

func main() {
	fmt.Printf(
		"Build version: %s\nBuild date: %s\nBuild commit: %s\n",
		buildVersion,
		buildDate,
		buildCommit,
	)

	clientApp, err := client.NewApp(defaultGrpcAddress, defaultDebug)
	if err != nil {
		panic(err)
	}
	defer clientApp.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		s := <-c
		fmt.Println("Signal received: " + s.String())
		cancel()
	}()
	err = clientApp.Run(ctx)
	if err != nil {
		panic(err)
	}
}
