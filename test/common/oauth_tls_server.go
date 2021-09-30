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
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/gbrlsnchs/jwt"
)

type OAuthTLSServer struct {
	BaseURL string

	Server     *httptest.Server
	Router     *mux.Router
	privateKey *rsa.PrivateKey // public key privateKey.PublicKey
	signer     jwt.Signer
	keyID      string

	mutex                     *sync.RWMutex
	tokenKeysRequestCallCount int
}

func NewOAuthTLSServer() *OAuthTLSServer {
	privateKey := generatePrivateKey()

	os := &OAuthTLSServer{
		privateKey: privateKey,
		signer:     jwt.RS256(privateKey, &privateKey.PublicKey),
		keyID:      randomName("key"),
		Router:     mux.NewRouter(),
		mutex:      &sync.RWMutex{},
	}
	os.Router.HandleFunc("/.well-known/openid-configuration", os.getOpenIDConfig)
	os.Router.HandleFunc("/oauth/token", os.getToken)
	os.Router.HandleFunc("/token_keys", os.getTokenKeys)
	os.Start()

	return os
}

func (os *OAuthTLSServer) Start() {
	if os.Server != nil {
		panic("OAuth server already started")
	}
	os.Server = httptest.NewTLSServer(os.Router)
	os.BaseURL = os.Server.URL
}

func (os *OAuthTLSServer) Close() {
	if os != nil && os.Server != nil {
		os.Server.Close()
		os.Server = nil
	}
}

func (os *OAuthTLSServer) URL() string {
	return os.BaseURL
}

func (os *OAuthTLSServer) TokenKeysRequestCallCount() int {
	return os.tokenKeysRequestCallCount
}

func (os *OAuthTLSServer) getOpenIDConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{
		 "issuer": "` + os.BaseURL + `/oauth/token",
		 "jwks_uri": "` + os.BaseURL + `/token_keys"
	 }`))
}

func (os *OAuthTLSServer) RotateTokenKey() {
	os.keyID = randomName("key")
	privateKey := generatePrivateKey()
	os.privateKey = privateKey
	os.signer = jwt.RS256(privateKey, &privateKey.PublicKey)
}

func (os *OAuthTLSServer) getToken(w http.ResponseWriter, r *http.Request) {
	token := os.CreateToken(map[string]interface{}{
		"user_name": "testUser",
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"access_token": "` + token + `"}`))
}

func (os *OAuthTLSServer) CreateToken(payload map[string]interface{}) string {
	var issuerURL string
	if iss, ok := payload["iss"]; ok {
		issuerURL = iss.(string)
	} else {
		issuerURL = os.BaseURL + "/oauth/token"
	}
	nextYear := time.Now().Add(365 * 24 * time.Hour)
	token, err := jwt.Sign(os.signer, &jwt.Options{
		Issuer:         issuerURL,
		KeyID:          os.keyID,
		Audience:       "sm",
		Subject:        "test-user",
		ExpirationTime: nextYear,
		Public:         payload,
	})
	if err != nil {
		panic(err)
	}
	return token
}

func (os *OAuthTLSServer) getTokenKeys(w http.ResponseWriter, r *http.Request) {
	jwk := newJwkResponse(os.keyID, os.privateKey.PublicKey)
	responseBody, _ := json.Marshal(&struct {
		Keys []jwkResponse `json:"keys"`
	}{
		Keys: []jwkResponse{*jwk},
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)

	os.mutex.Lock()
	os.tokenKeysRequestCallCount++
	os.mutex.Unlock()
}
