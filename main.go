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
	//os.Setenv("SM_SERVER_PORT", "8181")
	//os.Setenv("SM_SERVER_REQUESTTIMEOUT", "5")
	//os.Setenv("SM_SERVER_SHUTDOWNTIMEOUT", "6")
	//os.Setenv("SM_LOG_LEVEL", "error")
	//os.Setenv("SM_LOG_FORMAT", "text")
	//os.Setenv("SM_DB_URI", "postgres://postgres:postgres@localhost:5555/postgres?sslmode=disable")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handleInterrupts(ctx, cancel)

	config, err := server.NewConfiguration(env.Default())
	if err != nil {
		logrus.Fatal("Error loading configuration: ", err)
	}


	storage, err := storage.Use(ctx, postgres.Storage, config.DbURI)
	if err != nil {
		logrus.Fatal("Error creating storage: ", err)
	}
	defaultAPI := api.Default(storage)

	srv, err := server.New(defaultAPI, config)
	if err != nil {
		logrus.Fatal("Error creating new server: ", err)
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
