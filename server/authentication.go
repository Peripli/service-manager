package server

import (
	"errors"
	"net/http"

	"github.com/Peripli/service-manager/rest"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func authenticationMiddleware(smUsername, smPassword string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

			username, password, authOK := request.BasicAuth()
			if !authOK {
				logrus.Debug("No authorization with request")

				sendUnauthorizedJSONResponse(response, rest.CreateErrorResponse(
					errors.New("No authorization method used with request"),
					http.StatusUnauthorized,
					"MissingAuthorization"))
				return
			}

			if username != smUsername || password != smPassword {
				logrus.Debug("Invalid credentials provided")

				sendUnauthorizedJSONResponse(response, rest.CreateErrorResponse(
					errors.New("The supplied credentials could not be authorized"),
					http.StatusUnauthorized,
					"InvalidCredentials"))
				return
			}

			next.ServeHTTP(response, request)
		})
	}
}

func sendUnauthorizedJSONResponse(response http.ResponseWriter, err error) {
	response.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	rest.SendJSON(response, http.StatusUnauthorized, err)
}
