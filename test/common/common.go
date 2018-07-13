/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package common

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/Peripli/service-manager/app"
	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/types"
	"github.com/gavv/httpexpect"
	"github.com/gbrlsnchs/jwt"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/sirupsen/logrus"
)

type Object = map[string]interface{}
type Array = []interface{}

const Catalog = `{
  "services": [
    {
      "bindable": true,
      "description": "service",
      "id": "98418a7a-002e-4ff9-b66a-d03fc3d56b16",
      "metadata": {
        "displayName": "test",
        "longDescription": "test"
      },
      "name": "test",
      "plan_updateable": false,
      "plans": [
        {
          "description": "test",
          "free": true,
          "id": "9bb3b29e-bbf9-4900-b926-2f8e9c9a3347",
          "metadata": {
            "bullets": [
              "Plan with basic functionality and relaxed security, excellent for development and try-out purposes"
            ],
            "displayName": "lite"
          },
          "name": "lite"
        }
      ],
      "tags": [
        "test"
      ]
    }
  ]
}`

func GetServerHandler(api *rest.API, tokenIssuerURL string) http.Handler {
	set := env.EmptyFlagSet()
	config.AddPFlags(set)
	set.Set("file.location", "./test/common")

	serverEnv, err := env.New(set)
	if err != nil {
		logrus.Fatal("Error creating server: ", err)
	}
	cfg, err := config.New(serverEnv)

	params := &app.Parameters{
		Settings: cfg,
		API:      api,
	}
	if tokenIssuerURL != "" {
		cfg.API.TokenIssuerURL = tokenIssuerURL
	}
	srv, err := app.New(context.Background(), params)
	if err != nil {
		logrus.Fatal("Error creating server router during test server initialization: ", err)
	}
	return srv.Handler
}

func MapContains(actual Object, expected Object) {
	for k, v := range expected {
		value, ok := actual[k]
		if !ok {
			Fail(fmt.Sprintf("Missing property '%s'", k), 1)
		}
		if value != v {
			Fail(
				fmt.Sprintf("For property '%s':\nExpected: %s\nActual: %s", k, v, value),
				1)
		}
	}
}

func RemoveAllBrokers(SM *httpexpect.Expect) {
	removeAll(SM, "brokers", "/v1/service_brokers")
}

func RemoveAllPlatforms(SM *httpexpect.Expect) {
	removeAll(SM, "platforms", "/v1/platforms")
}

func removeAll(SM *httpexpect.Expect, entity, rootURLPath string) {
	By("removing all " + entity)
	resp := SM.GET(rootURLPath).
		Expect().Status(http.StatusOK).JSON().Object()
	for _, val := range resp.Value(entity).Array().Iter() {
		id := val.Object().Value("id").String().Raw()
		SM.DELETE(rootURLPath + "/" + id).
			Expect().Status(http.StatusOK)
	}
}

func RegisterBroker(brokerJSON Object, SM *httpexpect.Expect) string {
	reply := SM.POST("/v1/service_brokers").
		WithJSON(brokerJSON).
		Expect().Status(http.StatusCreated).JSON().Object()
	return reply.Value("id").String().Raw()
}

func RegisterPlatform(platformJSON Object, SM *httpexpect.Expect) *types.Platform {
	reply := SM.POST("/v1/platforms").
		WithJSON(platformJSON).
		Expect().Status(http.StatusCreated).JSON().Object().Raw()

	platform := &types.Platform{}
	mapstructure.Decode(reply, platform)
	return platform
}

func setResponse(rw http.ResponseWriter, status int, message string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	rw.Write([]byte(message))
}

func NewValidBrokerRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/v2/catalog", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, Catalog)
	})

	router.HandleFunc("/v2/service_instances/{instance_id}", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusCreated, "{}")
	}).Methods("PUT")

	router.HandleFunc("/v2/service_instances/{instance_id}", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, "{}")
	}).Methods("DELETE")

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(rw http.ResponseWriter, req *http.Request) {
		response := fmt.Sprintf(`{"credentials": {"instance_id": "%s" , "binding_id": "%s"}}`, mux.Vars(req)["instance_id"], mux.Vars(req)["binding_id"])
		setResponse(rw, http.StatusCreated, response)
	}).Methods("PUT")

	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, "{}")
	}).Methods("DELETE")

	return router
}

func NewFailingBrokerRouter() *mux.Router {
	router := mux.NewRouter()

	router.PathPrefix("/v2/catalog").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, Catalog)
	})

	router.PathPrefix("/").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusNotAcceptable, `{"description": "expected error"}`)
	})

	return router
}

func generatePrivateKey() *rsa.PrivateKey {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	return privateKey
}

type jwkResponse struct {
	KeyType   string `json:"kty"`
	Use       string `json:"sig"`
	KeyID     string `json:"kid"`
	Algorithm string `json:"alg"`
	Value     string `json:"value"`

	PublicKeyExponent string `json:"e"`
	PublicKeyModulus  string `json:"n"`
}

func newJwkResponse(keyID string, publicKey rsa.PublicKey) *jwkResponse {
	modulus := base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())

	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, uint32(publicKey.E))
	bytes = bytes[:3]
	exponent := base64.RawURLEncoding.EncodeToString(bytes)

	return &jwkResponse{
		KeyType:           "RSA",
		Use:               "sig",
		KeyID:             keyID,
		Algorithm:         "RSA256",
		Value:             "",
		PublicKeyModulus:  modulus,
		PublicKeyExponent: exponent,
	}
}

func RequestToken(issuerURL string) string {
	issuer := httpexpect.New(GinkgoT(), issuerURL)
	token := issuer.GET("/oauth/token").Expect().
		Status(http.StatusOK).JSON().Object().
		Value("access_token").String().Raw()
	return token
}

func SetupMockOAuthServer() *httptest.Server {
	privateKey := generatePrivateKey()
	publicKey := privateKey.PublicKey
	signer := jwt.RS256(privateKey, &publicKey)
	keyID := "test-key"

	var issuerURL string

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"issuer": "` + issuerURL + `/oauth/token",
			"jwks_uri": "` + issuerURL + `/token_keys"
		}`))
	})

	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		nextYear := time.Now().Add(24 * 30 * 12 * time.Hour)
		token, err := jwt.Sign(signer, &jwt.Options{
			Issuer:         issuerURL + "/oauth/token",
			KeyID:          keyID,
			Audience:       "smctl",
			ExpirationTime: nextYear})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token": "` + token + `"}`))
	})

	mux.HandleFunc("/token_keys", func(w http.ResponseWriter, r *http.Request) {
		jwk := newJwkResponse(keyID, publicKey)
		responseBody, _ := json.Marshal(&struct {
			Keys []jwkResponse `json:"keys"`
		}{
			Keys: []jwkResponse{*jwk},
		})

		w.Header().Set("Content-Type", "application/json")
		w.Write(responseBody)
	})

	server := httptest.NewServer(mux)
	issuerURL = server.URL

	return server
}

func MakeBroker(name string, url string, description string) Object {
	return Object{
		"name":        name,
		"broker_url":  url,
		"description": description,
		"credentials": Object{
			"basic": Object{
				"username": "buser",
				"password": "bpass",
			},
		},
	}
}

func MakePlatform(id string, name string, atype string, description string) Object {
	return Object{
		"id":          id,
		"name":        name,
		"type":        atype,
		"description": description,
	}
}

func FakeBrokerServer(code *int, response interface{}) *ghttp.Server {
	brokerServer := ghttp.NewServer()
	brokerServer.RouteToHandler(http.MethodGet, regexp.MustCompile(".*"), ghttp.RespondWithPtr(code, response))
	return brokerServer
}

func VerifyReqReceived(server *ghttp.Server, times int, method, path string, rawQuery ...string) {
	timesReceived := 0
	for _, req := range server.ReceivedRequests() {
		if req.Method == method && strings.Contains(req.URL.Path, path) {
			if len(rawQuery) == 0 {
				timesReceived++
				continue
			}
			values, err := url.ParseQuery(rawQuery[0])
			Expect(err).ShouldNot(HaveOccurred())
			if reflect.DeepEqual(req.URL.Query(), values) {
				timesReceived++
			}
		}
	}
	if times != timesReceived {
		Fail(fmt.Sprintf("Request with method = %s, path = %s, rawQuery = %s expected to be received atleast "+
			"%d times but was received %d times", method, path, rawQuery, times, timesReceived))
	}
}

func VerifyBrokerCatalogEndpointInvoked(server *ghttp.Server, times int) {
	VerifyReqReceived(server, times, http.MethodGet, "/v2/catalog")
}

func ClearReceivedRequests(code *int, response interface{}, server *ghttp.Server) {
	server.Reset()
	server.RouteToHandler(http.MethodGet, regexp.MustCompile(".*"), ghttp.RespondWithPtr(code, response))
}
