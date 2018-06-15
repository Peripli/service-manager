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
	"github.com/spf13/pflag"
)

func init() {
	pflag.String("api_token_issuer_url", "alabala", "authz token issuer url")
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handleInterrupts(ctx, cancel)

	srv, err := sm.New(ctx, cf.NewEnv(config.NewEnv("SM")))
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
