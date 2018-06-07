package main

import (
	"context"
	"flag"

	"os"
	"os/signal"

	cfenv "github.com/Peripli/service-manager/cf/env"
	"github.com/Peripli/service-manager/env"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/sm"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handleInterrupts(ctx, cancel)

	srv, err := sm.NewServer(ctx, getEnvironment())
	if err != nil {
		logrus.Fatal("Error creating the server: ", err)
	}
	srv.Run(ctx)
}

func handleInterrupts(ctx context.Context, cancelFunc context.CancelFunc) {
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt)
	go func() {
		select {
		case <-term:
			logrus.Error("Received OS interrupt, exiting gracefully...")
			cancelFunc()
		case <-ctx.Done():
			return
		}
	}()
}

func getEnvironment() server.Environment {
	var configFileLocation string
	flag.StringVar(&configFileLocation, "config_location", ".", "Location of the application.yaml file")
	flag.Parse()

	logrus.Debugf("config_location: %s", configFileLocation)

	runEnvironment := env.New(&env.ConfigFile{
		Path:   configFileLocation,
		Name:   "application",
		Format: "yaml",
	}, "SM")

	if _, exists := os.LookupEnv("VCAP_APPLICATION"); exists {
		return cfenv.New(runEnvironment)
	}
	return runEnvironment
}
