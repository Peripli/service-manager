package oidc

import (
	"fmt"
	"github.com/Peripli/service-manager/test/tls_settings"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Peripli/service-manager/pkg/auth"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestAuthStrategy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "")
}

var _ = Describe("Service Manager Auth strategy test", func() {
	var authStrategy auth.Authenticator
	var authOptions *auth.Options
	var configurationResponseCode int
	var configurationResponseBody []byte
	var responseStatusCode int
	var responseBody []byte
	var uaaServer *httptest.Server

	createUAAHandler := func() http.HandlerFunc {
		return func(response http.ResponseWriter, req *http.Request) {
			response.Header().Add("Content-Type", "application/json")
			if strings.Contains(req.URL.String(), ".well-known/openid-configuration") {
				response.WriteHeader(configurationResponseCode)
				response.Write(configurationResponseBody)
			} else {
				response.WriteHeader(responseStatusCode)
				response.Write([]byte(responseBody))
			}
		}
	}

	BeforeSuite(func() {
		uaaServer = httptest.NewServer(createUAAHandler())
		configurationResponseCode = http.StatusOK
		configurationResponseBody = []byte(`{"token_endpoint": "` + uaaServer.URL + `"}`)

		authStrategy, authOptions, _ = NewOpenIDStrategy(&auth.Options{
			IssuerURL: uaaServer.URL,
		})

		Expect(authOptions).To(Equal(&auth.Options{
			IssuerURL:             uaaServer.URL,
			TokenEndpoint:         uaaServer.URL,
			AuthorizationEndpoint: "",
		}))
	})

	AfterSuite(func() {
		if uaaServer != nil {
			uaaServer.Close()
		}
	})

	BeforeEach(func() {
		configurationResponseCode = http.StatusOK
		configurationResponseBody = []byte(`{"token_endpoint": "` + uaaServer.URL + `"}`)
	})

	Describe("", func() {
		Context("when configuration response is invalid", func() {
			It("should handle wrong response code", func() {
				configurationResponseCode = http.StatusNotFound
				_, _, err := NewOpenIDStrategy(&auth.Options{
					IssuerURL: uaaServer.URL,
				})

				Expect(err).Should(HaveOccurred())
				Expect(err).To(MatchError("error occurred while fetching openid configuration: unexpected status code"))
			})

			It("should handle wrong JSON body", func() {
				configurationResponseCode = http.StatusOK
				configurationResponseBody = []byte(`{"}`)
				_, _, err := NewOpenIDStrategy(&auth.Options{
					IssuerURL: uaaServer.URL,
				})

				Expect(err).Should(HaveOccurred())
			})
		})
	})

	Describe("token generation", func() {
		Context("when valid username and password are used", func() {
			It("should issue token", func() {
				responseStatusCode = http.StatusOK
				responseBody = []byte(`{
					"access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleS0xIiwidHlwIjoiSldUIn0.eyJqdGkiOiIzNWFjNDZkZGI0NjQ0YzEyODA1MGI1MDhmOTg3N2M5MSIsInN1YiI6ImYwYmYzNzA1LWMxNWMtNDYxOS1iMzkyLTg2YWYzODRlODkxNiIsInNjb3BlIjpbIm5ldHdvcmsud3JpdGUiLCJjbG91ZF9jb250cm9sbGVyLmFkbWluIiwicm91dGluZy5yb3V0ZXJfZ3JvdXBzLnJlYWQiLCJjbG91ZF9jb250cm9sbGVyLndyaXRlIiwibmV0d29yay5hZG1pbiIsImRvcHBsZXIuZmlyZWhvc2UiLCJvcGVuaWQiLCJyb3V0aW5nLnJvdXRlcl9ncm91cHMud3JpdGUiLCJzY2ltLnJlYWQiLCJ1YWEudXNlciIsImNsb3VkX2NvbnRyb2xsZXIucmVhZCIsInBhc3N3b3JkLndyaXRlIiwic2NpbS53cml0ZSJdLCJjbGllbnRfaWQiOiJjZiIsImNpZCI6ImNmIiwiYXpwIjoiY2YiLCJncmFudF90eXBlIjoicGFzc3dvcmQiLCJ1c2VyX2lkIjoiZjBiZjM3MDUtYzE1Yy00NjE5LWIzOTItODZhZjM4NGU4OTE2Iiwib3JpZ2luIjoidWFhIiwidXNlcl9uYW1lIjoiYWRtaW4iLCJlbWFpbCI6ImFkbWluIiwiYXV0aF90aW1lIjoxNTI3NzU3MjMzLCJyZXZfc2lnIjoiYTRiYWI4MTQiLCJpYXQiOjE1Mjc3NTcyMzMsImV4cCI6MTUyNzc1NzgzMywiaXNzIjoiaHR0cHM6Ly91YWEubG9jYWwucGNmZGV2LmlvL29hdXRoL3Rva2VuIiwiemlkIjoidWFhIiwiYXVkIjpbImNsb3VkX2NvbnRyb2xsZXIiLCJzY2ltIiwicGFzc3dvcmQiLCJjZiIsInVhYSIsIm9wZW5pZCIsImRvcHBsZXIiLCJuZXR3b3JrIiwicm91dGluZy5yb3V0ZXJfZ3JvdXBzIl19.Srd_204A3KyHAQ2QibxwxhRm6mwVRRdkJLluiOua6KHmj_x8LLLu6XA9G1e5LNzW_hNqmwxi1fUeFU7NsfUudo46r6pcdfMT0yl7x0qUdizKKZNSkRsoB3BBn1aTBMAgAtc_VBRC8KWCL6Sdy2V0zJ4C-D2nqnYu9vmsK1_tSao",
					"token_type": "bearer",
					"refresh_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleS0xIiwidHlwIjoiSldUIn0.eyJqdGkiOiJlNTI2ZDZmNmI4ODk0YjJkOTNhYjI5YTlhY2NmOGNhOS1yIiwic3ViIjoiZjBiZjM3MDUtYzE1Yy00NjE5LWIzOTItODZhZjM4NGU4OTE2Iiwic2NvcGUiOlsibmV0d29yay53cml0ZSIsImNsb3VkX2NvbnRyb2xsZXIuYWRtaW4iLCJyb3V0aW5nLnJvdXRlcl9ncm91cHMucmVhZCIsImNsb3VkX2NvbnRyb2xsZXIud3JpdGUiLCJuZXR3b3JrLmFkbWluIiwiZG9wcGxlci5maXJlaG9zZSIsIm9wZW5pZCIsInJvdXRpbmcucm91dGVyX2dyb3Vwcy53cml0ZSIsInNjaW0ucmVhZCIsInVhYS51c2VyIiwiY2xvdWRfY29udHJvbGxlci5yZWFkIiwicGFzc3dvcmQud3JpdGUiLCJzY2ltLndyaXRlIl0sImlhdCI6MTUyNzc1NzIzMywiZXhwIjoxNTMwMzQ5MjMzLCJjaWQiOiJjZiIsImNsaWVudF9pZCI6ImNmIiwiaXNzIjoiaHR0cHM6Ly91YWEubG9jYWwucGNmZGV2LmlvL29hdXRoL3Rva2VuIiwiemlkIjoidWFhIiwiZ3JhbnRfdHlwZSI6InBhc3N3b3JkIiwidXNlcl9uYW1lIjoiYWRtaW4iLCJvcmlnaW4iOiJ1YWEiLCJ1c2VyX2lkIjoiZjBiZjM3MDUtYzE1Yy00NjE5LWIzOTItODZhZjM4NGU4OTE2IiwicmV2X3NpZyI6ImE0YmFiODE0IiwiYXVkIjpbImNsb3VkX2NvbnRyb2xsZXIiLCJzY2ltIiwicGFzc3dvcmQiLCJjZiIsInVhYSIsIm9wZW5pZCIsImRvcHBsZXIiLCJuZXR3b3JrIiwicm91dGluZy5yb3V0ZXJfZ3JvdXBzIl19.fNWVIyrjM7zIf89R1iMwKLNkBwE3Go51OKnnGnpONSsh0KciogcdEN9pYVSZMeb37bDmlc6L-wYUUCSY-ZP4VNm9pZtC-uhIfFy8kT6ZHADpp0IuNbD5AK48NC6yRs8Qgux8OV2UHryxlcMVfCC-EfUUaI6Mcz4JWh1EU7ojesM",
					"expires_in": 599,
					"scope": "network.write cloud_controller.admin routing.router_groups.read cloud_controller.write network.admin doppler.firehose openid routing.router_groups.write scim.read uaa.user cloud_controller.read password.write scim.write",
					"jti": "35ac46ddb4644c128050b508f9877c91"
				}`)

				token, err := authStrategy.PasswordCredentials("admin", "admin")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(token.AccessToken).To(Equal("eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleS0xIiwidHlwIjoiSldUIn0.eyJqdGkiOiIzNWFjNDZkZGI0NjQ0YzEyODA1MGI1MDhmOTg3N2M5MSIsInN1YiI6ImYwYmYzNzA1LWMxNWMtNDYxOS1iMzkyLTg2YWYzODRlODkxNiIsInNjb3BlIjpbIm5ldHdvcmsud3JpdGUiLCJjbG91ZF9jb250cm9sbGVyLmFkbWluIiwicm91dGluZy5yb3V0ZXJfZ3JvdXBzLnJlYWQiLCJjbG91ZF9jb250cm9sbGVyLndyaXRlIiwibmV0d29yay5hZG1pbiIsImRvcHBsZXIuZmlyZWhvc2UiLCJvcGVuaWQiLCJyb3V0aW5nLnJvdXRlcl9ncm91cHMud3JpdGUiLCJzY2ltLnJlYWQiLCJ1YWEudXNlciIsImNsb3VkX2NvbnRyb2xsZXIucmVhZCIsInBhc3N3b3JkLndyaXRlIiwic2NpbS53cml0ZSJdLCJjbGllbnRfaWQiOiJjZiIsImNpZCI6ImNmIiwiYXpwIjoiY2YiLCJncmFudF90eXBlIjoicGFzc3dvcmQiLCJ1c2VyX2lkIjoiZjBiZjM3MDUtYzE1Yy00NjE5LWIzOTItODZhZjM4NGU4OTE2Iiwib3JpZ2luIjoidWFhIiwidXNlcl9uYW1lIjoiYWRtaW4iLCJlbWFpbCI6ImFkbWluIiwiYXV0aF90aW1lIjoxNTI3NzU3MjMzLCJyZXZfc2lnIjoiYTRiYWI4MTQiLCJpYXQiOjE1Mjc3NTcyMzMsImV4cCI6MTUyNzc1NzgzMywiaXNzIjoiaHR0cHM6Ly91YWEubG9jYWwucGNmZGV2LmlvL29hdXRoL3Rva2VuIiwiemlkIjoidWFhIiwiYXVkIjpbImNsb3VkX2NvbnRyb2xsZXIiLCJzY2ltIiwicGFzc3dvcmQiLCJjZiIsInVhYSIsIm9wZW5pZCIsImRvcHBsZXIiLCJuZXR3b3JrIiwicm91dGluZy5yb3V0ZXJfZ3JvdXBzIl19.Srd_204A3KyHAQ2QibxwxhRm6mwVRRdkJLluiOua6KHmj_x8LLLu6XA9G1e5LNzW_hNqmwxi1fUeFU7NsfUudo46r6pcdfMT0yl7x0qUdizKKZNSkRsoB3BBn1aTBMAgAtc_VBRC8KWCL6Sdy2V0zJ4C-D2nqnYu9vmsK1_tSao"))
			})
		})

		Context("when token response is invalid", func() {
			It("should handle wrong response code", func() {
				errorMsg := "missing client_id or client_secret"
				responseStatusCode = http.StatusBadRequest
				responseBody = []byte(fmt.Sprintf(`{"error_description": "%s"}`, errorMsg))
				_, err := authStrategy.PasswordCredentials("admin", "admin")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errorMsg))
			})

			It("should handle response that is not JSON", func() {
				responseStatusCode = http.StatusOK
				responseStatusCode = http.StatusBadRequest
				errorMsg := "internal error"
				responseBody = []byte(errorMsg)
				_, err := authStrategy.PasswordCredentials("admin", "admin")

				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errorMsg))
			})
		})
	})

	Describe("OIDC Client", func() {
		newToken := func(validity time.Duration) *auth.Token {
			return &auth.Token{
				AccessToken:  "access-token",
				TokenType:    "token-type",
				RefreshToken: "refresh-token",
				ExpiresIn:    time.Now().Add(validity),
			}
		}
		options := &auth.Options{
			ClientID:      "client-id",
			ClientSecret:  "client-secret",
			User:          "user",
			Password:      "password",
			TokenEndpoint: "http://token-endpoint",
		}
		token := newToken(1 * time.Hour)
		tokenNoRefreshToken := &auth.Token{
			AccessToken:  "access-token",
			TokenType:    "token-type",
			RefreshToken: "",
			ExpiresIn:    time.Now().Add(-1 * time.Hour),
		}

		DescribeTable("mTLS oidc client",
			func(options *auth.Options, expectedErrMsg string) {
				client, err := NewClient(options, token)

				if expectedErrMsg != "" {
					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring(expectedErrMsg))
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(err).To(BeNil())
					Expect(client).ToNot(BeNil())
				}
			},
			Entry("mTLS with valid certificate & key",
				&auth.Options{
					Certificate:   tls_settings.ClientCertificate,
					Key:           tls_settings.ClientKey,
					TokenEndpoint: "http://token-endpoint",
					ClientID:      "client-id",
				},
				""),
			Entry("mTLS invalid certificate - returns error",
				&auth.Options{
					Certificate:   "certificate",
					Key:           tls_settings.ClientKey,
					TokenEndpoint: "http://token-endpoint",
					ClientID:      "client-id",
				},
				"tls: failed to find any PEM data in certificate input"),
			Entry("mTLS invalid certificate file - returns error",
				&auth.Options{
					Certificate:   "certificate.pem",
					Key:           "key.pem",
					TokenEndpoint: "http://token-endpoint",
					ClientID:      "client-id",
				},
				"no such file or directory"),
		)
		DescribeTable("NewClient",
			func(options *auth.Options, token *auth.Token, expectedErrMsg string, expetedToken *auth.Token) {
				client, err := NewClient(options, token)
				Expect(err).ToNot(HaveOccurred())

				t, err := client.Token()
				if expectedErrMsg == "" {
					Expect(err).To(BeNil())
					Expect(*t).To(Equal(*token))
				} else {
					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring(expectedErrMsg))
				}
			},
			Entry("Valid token - reuses the token", options, token, "", token),
			Entry("No client credentials and valid token - reuses the token",
				&auth.Options{}, token, "", token),
			Entry("No client credentials and expired token - returns error to login",
				&auth.Options{},
				newToken(-1*time.Hour),
				"access token has expired",
				nil),
			Entry("With client credentials and refresh token - refreshes the token",
				options,
				newToken(-1*time.Hour),
				options.TokenEndpoint,
				nil),
			Entry("With client credentials and no refresh token - fetches a new token using client credentials flow",
				&auth.Options{
					ClientID:      "client-id",
					ClientSecret:  "client-secret",
					TokenEndpoint: "http://token-endpoint",
				},
				tokenNoRefreshToken,
				"http://token-endpoint",
				nil),
			Entry("With client and user credentials and no refresh token - returns error to login",
				options,
				tokenNoRefreshToken,
				"access token has expired",
				nil),
		)
	})
})
