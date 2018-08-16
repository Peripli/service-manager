package util

import (
	"context"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"
)

func HandleInterrupts() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt)
	go func() {
		select {
		case <-term:
			logrus.Error("Received OS interrupt, exiting gracefully...")
			cancel()
		case <-ctx.Done():
			return
		}
	}()
	return ctx, cancel
}
