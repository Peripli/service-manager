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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gbrlsnchs/jwt"
)

type OAuthServer struct {
	server *httptest.Server
}

func NewOAuthServer() *OAuthServer {
	privateKey := generatePrivateKey()
	publicKey := privateKey.PublicKey
	signer := jwt.RS256(privateKey, &publicKey)
	keyID := "test-key"

	var issuerURL string
	mux := http.NewServeMux()

	server := httptest.NewServer(mux)
	issuerURL = server.URL

	return &OAuthServer{
		server: server,
	}
}

func (os *OAuthServer) Close() {
	os.server.Close()
}

func SetupFakeOAuthServer() *httptest.Server {

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

	return server
}
