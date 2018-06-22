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
	flags := initFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handleInterrupts(ctx, cancel)

	srv, err := sm.NewServer(ctx, getEnvironment(flags))
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

func initFlags() map[string]interface{} {
	configFileLocation := flag.String("config_location", ".", "Location of the application.yml file")
	flag.Parse()
	return map[string]interface{}{"config_location": *configFileLocation}
}

func getEnvironment(flags map[string]interface{}) server.Environment {
	configFileLocation := flags["config_location"].(string)
	logrus.Infof("config_location: %s", configFileLocation)

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
