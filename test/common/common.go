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

	"bytes"
	"io"
	"io/ioutil"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gavv/httpexpect"
	"github.com/gbrlsnchs/jwt"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
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
		Expect().JSON().Object()
	for _, val := range resp.Value(entity).Array().Iter() {
		id := val.Object().Value("id").String().Raw()
		SM.DELETE(rootURLPath + "/" + id).Expect()
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

func setResponse(rw http.ResponseWriter, status int, message, brokerID string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.Header().Set("X-Broker-ID", brokerID)
	rw.WriteHeader(status)
	rw.Write([]byte(message))
}

func SetupFakeServiceBrokerServerWithPrefix(brokerID, prefix string) *httptest.Server {
	router := mux.NewRouter()

	router.HandleFunc(prefix+"/v2/catalog", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, Catalog, brokerID)
	})

	router.HandleFunc(prefix+"/v2/service_instances/{instance_id}", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusCreated, "{}", brokerID)
	}).Methods("PUT")

	router.HandleFunc(prefix+"/v2/service_instances/{instance_id}", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, "{}", brokerID)
	}).Methods("DELETE")

	router.HandleFunc(prefix+"/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(rw http.ResponseWriter, req *http.Request) {
		response := fmt.Sprintf(`{"credentials": {"instance_id": "%s" , "binding_id": "%s"}}`, mux.Vars(req)["instance_id"], mux.Vars(req)["binding_id"])
		setResponse(rw, http.StatusCreated, response, brokerID)
	}).Methods("PUT")

	router.HandleFunc(prefix+"/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, "{}", brokerID)
	}).Methods("DELETE")

	router.HandleFunc(prefix+"/v2/service_instances/{instance_id}/last_operation", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, `{"state": "succeeded"}`, brokerID)
	}).Methods("GET")

	router.HandleFunc(prefix+"/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, `{"state": "succeeded"}`, brokerID)
	}).Methods("GET")

	server := httptest.NewServer(router)
	if prefix != "" {
		server.URL = server.URL + prefix
	}

	return server
}

func SetupFakeServiceBrokerServer(brokerID string) *httptest.Server {
	return SetupFakeServiceBrokerServerWithPrefix(brokerID, "")
}

func SetupFakeFailingBrokerServer(brokerID string) *httptest.Server {
	router := mux.NewRouter()

	router.PathPrefix("/v2/catalog").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, Catalog, brokerID)
	})

	router.PathPrefix("/").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusNotAcceptable, `{"description": "expected error"}`, brokerID)
	})

	return httptest.NewServer(router)
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

func SetupFakeOAuthServer() *httptest.Server {
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
			Audience:       "sm",
			ExpirationTime: nextYear,
			Public: map[string]interface{}{
				"user_name": "testUser",
			},
		})
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

type HTTPReaction struct {
	Status int
	Body   string
	Err    error
}

type HTTPExpectations struct {
	URL     string
	Body    string
	Params  map[string]string
	Headers map[string]string
}

type NopCloser struct {
	io.Reader
}

func (NopCloser) Close() error { return nil }

func Closer(s string) io.ReadCloser {
	return NopCloser{bytes.NewBufferString(s)}
}

func DoHTTP(reaction *HTTPReaction, checks *HTTPExpectations) func(*http.Request) (*http.Response, error) {
	return func(request *http.Request) (*http.Response, error) {
		if checks != nil {
			if len(checks.URL) > 0 && !strings.Contains(checks.URL, request.URL.Host) {
				Fail(fmt.Sprintf("unexpected URL; expected %v, got %v", checks.URL, request.URL.Path))
			}

			for k, v := range checks.Headers {
				actualValue := request.Header.Get(k)
				if e, a := v, actualValue; e != a {
					Fail(fmt.Sprintf("unexpected header value for key %q; expected %v, got %v", k, e, a))
				}
			}

			for k, v := range checks.Params {
				actualValue := request.URL.Query().Get(k)
				if e, a := v, actualValue; e != a {
					Fail(fmt.Sprintf("unexpected parameter value for key %q; expected %v, got %v", k, e, a))
				}
			}

			var bodyBytes []byte
			if request.Body != nil {
				var err error
				bodyBytes, err = ioutil.ReadAll(request.Body)
				if err != nil {
					Fail(fmt.Sprintf("error reading request Body bytes: %v", err))
				}
			}

			if e, a := checks.Body, string(bodyBytes); e != a {
				Fail(fmt.Sprintf("unexpected request Body: expected %v, got %v", e, a))
			}
		}
		return &http.Response{
			StatusCode: reaction.Status,
			Body:       Closer(reaction.Body),
		}, reaction.Err
	}
}
