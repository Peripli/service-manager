package main

import (
	"context"
	"fmt"

	"os"
	"os/signal"

	"github.com/Peripli/service-manager/app"
	"github.com/Peripli/service-manager/cf"
	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handleInterrupts(ctx, cancel)

	set := env.EmptyFlagSet()
	config.AddPFlags(set)

	env, err := env.New(set)
	if err != nil {
		panic(fmt.Sprintf("error loading environment: %s", err))
	}
	if err := cf.SetCFOverrides(env); err != nil {
		panic(fmt.Sprintf("error setting CF environment values: %s", err))
	}
	cfg, err := config.New(env)
	if err != nil {
		panic(fmt.Sprintf("error loading configuration: %s", err))
	}

	parameters := &app.Parameters{
		Settings: cfg,
	}
	srv, err := app.New(ctx, parameters)
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
