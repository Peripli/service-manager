package util

import (
	"context"
	"os"
	"os/signal"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
)

// HandleInterrupts handles process signal interrupts
func HandleInterrupts(ctx context.Context, cancel context.CancelFunc) {

	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt)
	go func() {
		select {
		case <-term:
			log.C(ctx).Error("Received OS interrupt, exiting gracefully...")
			cancel()
		case <-ctx.Done():
			return
		}
	}()
}
