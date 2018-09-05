package filters

import (
	"net/http"
	"runtime/debug"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/gorilla/mux"
)

// NewRecoveryMiddleware returns a standard mux middleware that provides panic recovery
func NewRecoveryMiddleware() mux.MiddlewareFunc {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					httpError := &util.HTTPError{
						StatusCode:  http.StatusInternalServerError,
						Description: "Internal Server Error",
					}
					util.WriteError(httpError, w)
					debug.PrintStack()
					log.C(r.Context()).Error(err)
				}
			}()
			handler.ServeHTTP(w, r)
		})
	}
}
