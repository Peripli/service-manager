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
	"github.com/onsi/ginkgo"

	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	mathrand "math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gofrs/uuid"

	"strings"

	"bytes"
	"io"
	"io/ioutil"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gavv/httpexpect"
	"github.com/mitchellh/mapstructure"
	. "github.com/onsi/ginkgo"
)

type Object = map[string]interface{}
type Array = []interface{}

func RmNonNumbericFieldNames(obj Object) Object {
	for k, v := range obj {
		if _, err := strconv.Atoi(fmt.Sprintf("%v", v)); err == nil {
			continue
		}
		if _, err := strconv.ParseFloat(fmt.Sprintf("%v", v), 64); err != nil {
			delete(obj, k)
		}
	}
	return obj
}

func RmNumbericFieldNames(obj Object) Object {
	for k, v := range obj {
		if _, err := strconv.Atoi(fmt.Sprintf("%v", v)); err == nil {
			delete(obj, k)
		}
		if _, err := strconv.ParseFloat(fmt.Sprintf("%v", v), 64); err == nil {
			delete(obj, k)
		}
	}
	return obj
}

func RmNonJSONFieldNames(obj Object) Object {
	o := CopyObject(obj)

	for k, v := range o {
		isJSON := false
		if _, ok := v.(map[string]interface{}); ok {
			isJSON = true
		}
		if _, ok := v.([]interface{}); ok {
			isJSON = true
		}
		if !isJSON {
			delete(o, k)
		}
	}
	return o
}

func RmNotNullableFieldNames(obj Object, optionalFields []string) Object {
	o := CopyObject(obj)
	for objField := range o {
		found := false
		for _, field := range optionalFields {
			if field == objField {
				found = true
			}
		}
		if !found {
			delete(o, objField)
		}
	}
	return o
}

// TODO this can be used to leave out only the string fields and test corner cases with | in key and value, space and plus in the value and key, operator in the key
func RmNotStringFieldNames(obj Object) Object {
	panic("implement me")
}

//TODO more values of a type for multi right args case
func CopyObject(obj Object) Object {
	o := Object{}
	for k, v := range obj {
		o[k] = v
	}
	return o
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

func RemoveAllVisibilities(SM *httpexpect.Expect) {
	removeAll(SM, "visibilities", "/v1/visibilities")
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

func RegisterBrokerInSM(brokerJSON Object, SM *httpexpect.Expect) string {
	reply := SM.POST("/v1/service_brokers").
		WithJSON(brokerJSON).
		Expect().Status(http.StatusCreated).JSON().Object()
	return reply.Value("id").String().Raw()
}

func RegisterPlatformInSM(platformJSON Object, SM *httpexpect.Expect) *types.Platform {
	reply := SM.POST("/v1/platforms").
		WithJSON(platformJSON).
		Expect().Status(http.StatusCreated).JSON().Object().Raw()
	platform := &types.Platform{}
	mapstructure.Decode(reply, platform)
	return platform
}

func generatePrivateKey() *rsa.PrivateKey {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	return privateKey
}

func ExtractResourceIDs(entities []Object) []string {
	result := make([]string, 0, 0)
	if entities == nil {
		return result
	}
	for _, value := range entities {
		if _, ok := value["id"]; !ok {
			panic(fmt.Sprintf("No id found for test resource %v", value))
		}
		result = append(result, value["id"].(string))
	}
	return result
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

func MakePlatform(id string, name string, atype string, description string) Object {
	return Object{
		"id":          id,
		"name":        name,
		"type":        atype,
		"description": description,
	}
}

func MakeRandomizedPlatformWithNoDescription() Object {
	o := Object{}
	for _, key := range []string{"id", "name", "type"} {
		UUID, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}
		o[key] = UUID.String()

	}
	o["description"] = ""
	return o
}
func GenerateRandomPlatform() Object {
	o := Object{}
	for _, key := range []string{"id", "name", "type", "description"} {
		UUID, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}
		o[key] = UUID.String()

	}
	return o
}

//TODO this instead of By?
// prefix test logs somehow so they be differenciated from SM logs? SM to log only errors?
func Print(message string, args ...interface{}) {
	if len(args) == 0 {
		fmt.Fprint(ginkgo.GinkgoWriter, "\n"+message+"\n")
	} else {
		fmt.Fprintf(ginkgo.GinkgoWriter, "\n"+message+"\n", args)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandomName(prefix string) string {
	mathrand.Seed(time.Now().UnixNano())
	b := make([]rune, 15)
	for i := range b {
		b[i] = letterRunes[mathrand.Intn(len(letterRunes))]
	}
	return prefix + "-" + string(b)
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
			Request:    request,
		}, reaction.Err
	}
}

// e2e testovete
// definirame si custom describe DescribeWithExpectations i si definirame custom closure ot tipa na aftereach Expectations

// moje li po nqkakuv na4in da injectnem closure ot tipa na after each (AdditionalExpectations) prez e2e test suite-to

// moje v suite-to da ima expectations pool globalen i kogato se importne vsqko DescribeWithExpectations da tursi tam AdditionaleXPECTATIONS sus key stringa v desribe-a
