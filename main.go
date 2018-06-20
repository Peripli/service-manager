package main

import (
	"context"

	"os"
	"os/signal"

	"fmt"

	"github.com/Peripli/service-manager/cf"
	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/sm"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handleInterrupts(ctx, cancel)

	env := cf.NewEnv(config.NewEnv(config.File{
		Location: ".",
		Name:     "application",
		Format:   "yml",
	}))
	cfg, err := config.New(env)
	if err != nil {
		panic(fmt.Sprintf("error loading configuration: %s", err))
	}
	srv, err := sm.New(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("error creating SM server: %s", err))
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
