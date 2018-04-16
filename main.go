package main

import (
	"context"

	"os"
	"os/signal"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/env"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"
	"github.com/sirupsen/logrus"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handleInterrupts(ctx, cancel)

	config, err := server.NewConfiguration(env.Default())
	if err != nil {
		logrus.Fatal("Error loading configuration: ", err)
	}

	if err := config.Validate(); err != nil {
		logrus.Fatal("NewConfiguration: validation failed ", err)
	}

	storage, err := storage.Use(ctx, postgres.Storage, config.DbURI)
	if err != nil {
		logrus.Fatal("Error using storage: ", err)
	}
	defaultAPI := api.Default(storage)

	srv, err := server.New(defaultAPI, config)
	if err != nil {
		logrus.Fatal("Error creating server: ", err)
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
