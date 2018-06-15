package main

import (
	"context"

	"os"
	"os/signal"

	"github.com/Peripli/service-manager/cf"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/sm"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handleInterrupts(ctx, cancel)

	srv, err := sm.New(ctx, cf.NewEnv(server.NewEnv("SM")))

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
