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
	"github.com/spf13/pflag"

	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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

type closer interface {
	Close()
}

type urler interface {
	URL() string
}

type FakeServer interface {
	closer
	urler
}

type FlagValue struct {
	pflagValue pflag.Value
}

func (f FlagValue) Set(s string) error {
	return f.pflagValue.Set(s)
}

func (f FlagValue) String() string {
	return f.pflagValue.String()
}

func RemoveNonNumericArgs(obj Object) Object {
	return removeOnCondition(isNotNumeric, obj)
}

func RemoveNumericArgs(obj Object) Object {
	return removeOnCondition(isNumeric, obj)
}

func RemoveNonJSONArgs(obj Object) Object {
	return removeOnCondition(isNotJSON, obj)
}

func removeOnCondition(condition func(arg interface{}) bool, obj Object) Object {
	o := CopyObject(obj)

	for k, v := range o {
		if k == "labels" {
			labels := v.(map[string]interface{})
			for lKey, lValues := range labels {
				lVals := lValues.([]interface{})
				for index, lValue := range lVals {
					if condition(lValue) {
						labels[lKey] = append(lVals[:index], lVals[index+1:]...)
					}
				}
				if len(lVals) == 0 {
					delete(labels, lKey)
				}
			}
		} else if condition(v) {
			delete(o, k)
		}
	}
	return o
}

func isJson(arg interface{}) bool {
	if str, ok := arg.(string); ok {
		var jsonStr map[string]interface{}
		err := json.Unmarshal([]byte(str), &jsonStr)
		return err == nil
	}
	if _, ok := arg.(map[string]interface{}); ok {
		return true
	}
	if _, ok := arg.([]interface{}); ok {
		return true
	}
	return false
}

func isNotJSON(arg interface{}) bool {
	return !isJson(arg)
}

func isNumeric(arg interface{}) bool {
	if _, err := strconv.Atoi(fmt.Sprintf("%v", arg)); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(fmt.Sprintf("%v", arg), 64); err == nil {
		return true
	}
	return false
}

func isNotNumeric(arg interface{}) bool {
	return !isNumeric(arg)
}

func RemoveNotNullableFieldAndLabels(obj Object, objithMandatoryFields Object) Object {
	o := CopyObject(obj)
	for objField, objVal := range objithMandatoryFields {
		if str, ok := objVal.(string); ok && len(str) == 0 {
			//currently api returns empty string for nullable values
			continue
		}
		delete(o, objField)
	}

	delete(o, "labels")
	return o
}

func CopyObject(obj Object) Object {
	o := Object{}
	for k, v := range obj {
		if k == "labels" {
			l := map[string]interface{}{}
			for lKey, lValues := range v.(map[string]interface{}) {
				temp := []interface{}{}
				for _, v := range lValues.([]interface{}) {
					l[lKey] = append(temp, v)
				}
			}
			o[k] = l
		} else {
			o[k] = v
		}
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
	removeAll(SM, "service_brokers", "/v1/service_brokers")
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

func GenerateRandomBroker() Object {
	o := Object{}

	brokerServer := NewBrokerServer()
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	UUID2, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	o = Object{
		"name":        UUID.String(),
		"broker_url":  brokerServer.URL(),
		"description": UUID2.String(),
		"credentials": Object{
			"basic": Object{
				"username": brokerServer.Username,
				"password": brokerServer.Password,
			},
		},
	}
	return o
}

func Print(message string, args ...interface{}) {
	if len(args) == 0 {
		fmt.Fprint(ginkgo.GinkgoWriter, "\n"+message+"\n")
	} else {
		fmt.Fprintf(ginkgo.GinkgoWriter, "\n"+message+"\n", args)
	}
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
