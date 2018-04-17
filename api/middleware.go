package api

import (
	"log"
	"net/http"

	"github.com/sirupsen/logrus"
)

func authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		logrus.Debug()
		username, password, authOK := request.BasicAuth()
		if !authOK {
			logrus.Error("No authorization provided with request")

			response.WriteHeader(http.StatusUnauthorized)
			response.Write([]byte("Not authorized"))
			return
		}

		if username != "admin" || password != "admin" {
			logrus.Error("Invalid credentials provided")

			response.WriteHeader(http.StatusUnauthorized)
			response.Write([]byte("Invalid credentials"))
			return
		}
		log.Printf(">>>>>>> Authorization decoded:\nUsername: %s\nPassword: %s\n", username, password)
		next.ServeHTTP(response, request)
	})
}
