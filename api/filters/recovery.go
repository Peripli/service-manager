package filters

import (
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// NewRecoveryMiddleware returns a standard mux middleware that provides panic recovery
func NewRecoveryMiddleware() mux.MiddlewareFunc {
	return handlers.RecoveryHandler(
		handlers.PrintRecoveryStack(true),
		handlers.RecoveryLogger(&recoveryHandlerLogger{}),
	)
}

type recoveryHandlerLogger struct {
}

// PrintLn prints panic message and stack to error output
func (r *recoveryHandlerLogger) Println(args ...interface{}) {
	log.D().Errorln(args...)
}
