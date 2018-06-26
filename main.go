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

	set := config.SMFlagSet()
	config.AddPFlags(set)

	env, err := config.NewEnv(set)
	if err != nil {
		panic(fmt.Sprintf("error loading environment: %s", err))
	}
	if err := cf.SetEnvValues(env); err != nil {
		panic(fmt.Sprintf("error setting CF environment values: %s", err))
	}
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
