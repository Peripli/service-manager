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
	"time"

	"github.com/gbrlsnchs/jwt"
)

type OAuthServer struct {
	URL string

	server     *httptest.Server
	mux        *http.ServeMux
	privateKey *rsa.PrivateKey // public key privateKey.PublicKey
	signer     jwt.Signer
	keyID      string
}

func NewOAuthServer() *OAuthServer {
	privateKey := generatePrivateKey()

	os := &OAuthServer{
		privateKey: privateKey,
		signer:     jwt.RS256(privateKey, &privateKey.PublicKey),
		keyID:      "test-key",
		mux:        http.NewServeMux(),
	}
	os.mux.HandleFunc("/.well-known/openid-configuration", os.getOpenIDConfig)
	os.mux.HandleFunc("/oauth/token", os.getToken)
	os.mux.HandleFunc("/token_keys", os.getTokenKeys)

	return os
}

func (os *OAuthServer) Start() {
	if os.server != nil {
		panic("OAuth server already started")
	}
	os.server = httptest.NewServer(os.mux)
	os.URL = os.server.URL
}

func (os *OAuthServer) Close() {
	if os != nil && os.server != nil {
		os.server.Close()
		os.server = nil
	}
}

func (os *OAuthServer) getOpenIDConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{
		"issuer": "` + os.URL + `/oauth/token",
		"jwks_uri": "` + os.URL + `/token_keys"
	}`))
}

func (os *OAuthServer) getToken(w http.ResponseWriter, r *http.Request) {
	token := os.CreateToken(map[string]interface{}{
		"user_name": "testUser",
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"access_token": "` + token + `"}`))
}

func (os *OAuthServer) CreateToken(payload map[string]interface{}) string {
	nextYear := time.Now().Add(365 * 24 * time.Hour)
	token, err := jwt.Sign(os.signer, &jwt.Options{
		Issuer:         os.URL + "/oauth/token",
		KeyID:          os.keyID,
		Audience:       "sm",
		ExpirationTime: nextYear,
		Public:         payload,
	})
	if err != nil {
		panic(err)
	}
	return token
}

func (os *OAuthServer) getTokenKeys(w http.ResponseWriter, r *http.Request) {
	jwk := newJwkResponse(os.keyID, os.privateKey.PublicKey)
	responseBody, _ := json.Marshal(&struct {
		Keys []jwkResponse `json:"keys"`
	}{
		Keys: []jwkResponse{*jwk},
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}
