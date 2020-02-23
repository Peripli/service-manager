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
	"time"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"

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

func RemoveBooleanArgs(obj Object) Object {
	return removeOnCondition(isBoolean, obj)
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

func isBoolean(arg interface{}) bool {
	_, ok := arg.(bool)
	return ok
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

func CopyLabels(obj Object) Object {
	result := CopyObject(obj)
	return (result["labels"]).(Object)
}

func CopyFields(obj Object) Object {
	result := CopyObject(obj)
	delete(result, "labels")
	return result
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

func RemoveAllOperations(repository storage.TransactionalRepository) {
	removeAll(repository, types.OperationType)
}

func RemoveAllNotifications(repository storage.TransactionalRepository) {
	removeAll(repository, types.NotificationType)
}

func RemoveAllInstances(testCtx *TestContext) error {
	if err := testCtx.SMRepository.InTransaction(context.TODO(), func(ctx context.Context, storage storage.Repository) error {
		objectList, err := storage.List(context.TODO(), types.ServiceInstanceType)
		if err != nil {
			return err
		}
		for i := 0; i < objectList.Len(); i++ {
			instance := objectList.ItemAt(i).(*types.ServiceInstance)
			byID := query.ByField(query.EqualsOperator, "id", instance.ID)
			if err := storage.Delete(ctx, types.ServiceInstanceType, byID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		if err != util.ErrNotFoundInStorage {
			return err
		}
	}
	return nil
}

func RemoveAllBindings(testCtx *TestContext) error {
	if err := testCtx.SMRepository.InTransaction(context.TODO(), func(ctx context.Context, storage storage.Repository) error {
		objectList, err := storage.List(context.TODO(), types.ServiceBindingType)
		if err != nil {
			return err
		}
		for i := 0; i < objectList.Len(); i++ {
			binding := objectList.ItemAt(i).(*types.ServiceBinding)
			byID := query.ByField(query.EqualsOperator, "id", binding.ID)
			if err := storage.Delete(ctx, types.ServiceBindingType, byID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		if err != util.ErrNotFoundInStorage {
			return err
		}
	}

	return nil
}

func RemoveAllBrokers(repository storage.TransactionalRepository) {
	removeAll(repository, types.ServiceBrokerType)
}

func RemoveAllPlatforms(repository storage.TransactionalRepository) {
	removeAll(repository, types.PlatformType, query.ByField(query.NotEqualsOperator, "id", types.SMPlatform))
}

func RemoveAllVisibilities(repository storage.TransactionalRepository) {
	removeAll(repository, types.VisibilityType)
}

func removeAll(repository storage.TransactionalRepository, entity types.ObjectType, queries ...query.Criterion) {
	By("removing all " + entity.String())
	if err := repository.InTransaction(context.TODO(), func(ctx context.Context, storage storage.Repository) error {
		if len(queries) == 0 {
			if err := storage.Delete(ctx, entity); err != nil {
				return err
			}
		} else {
			if err := storage.Delete(ctx, entity, queries...); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		if err != util.ErrNotFoundInStorage {
			panic(err)
		}
	}
}

func RegisterBrokerInSM(brokerJSON Object, SM *SMExpect, headers map[string]string) Object {
	return SM.POST(web.ServiceBrokersURL).
		WithHeaders(headers).
		WithJSON(brokerJSON).Expect().Status(http.StatusCreated).JSON().Object().Raw()
}

func RegisterVisibilityForPlanAndPlatform(SM *SMExpect, planID, platformID string) string {
	return SM.POST(web.VisibilitiesURL).WithJSON(Object{
		"service_plan_id": planID,
		"platform_id":     platformID,
	}).Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
}

func CreateVisibilitiesForAllBrokerPlans(SM *SMExpect, brokerID string) {
	offerings := SM.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s'", brokerID)).Iter()
	offeringIDs := make([]string, 0, len(offerings))
	for _, offering := range offerings {
		offeringIDs = append(offeringIDs, offering.Object().Value("id").String().Raw())
	}
	plans := SM.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id in ('%s')", strings.Join(offeringIDs, "','"))).Iter()
	for _, p := range plans {
		SM.POST(web.VisibilitiesURL).WithJSON(Object{
			"service_plan_id": p.Object().Value("id").String().Raw(),
		}).Expect().Status(http.StatusCreated)
	}
}

func RegisterPlatformInSM(platformJSON Object, SM *SMExpect, headers map[string]string) *types.Platform {
	reply := SM.POST(web.PlatformsURL).
		WithHeaders(headers).
		WithJSON(platformJSON).
		Expect().Status(http.StatusCreated).JSON().Object().Raw()
	createdAtString := reply["created_at"].(string)
	updatedAtString := reply["updated_at"].(string)
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtString)
	if err != nil {
		panic(err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtString)
	if err != nil {
		panic(err)
	}
	platform := &types.Platform{
		Base: types.Base{
			ID:        reply["id"].(string),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Ready:     true,
		},
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: reply["credentials"].(map[string]interface{})["basic"].(map[string]interface{})["username"].(string),
				Password: reply["credentials"].(map[string]interface{})["basic"].(map[string]interface{})["password"].(string),
			},
		},
		Type:        reply["type"].(string),
		Description: reply["description"].(string),
		Name:        reply["name"].(string),
	}
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

	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(publicKey.E))
	data = data[:3]
	exponent := base64.RawURLEncoding.EncodeToString(data)

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

func GenerateRandomNotification() *types.Notification {
	uid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	return &types.Notification{
		Base: types.Base{
			ID:    uid.String(),
			Ready: true,
		},
		PlatformID: "",
		Resource:   "notification",
		Type:       "CREATED",
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
	var err error
	if len(args) == 0 {
		_, err = fmt.Fprint(ginkgo.GinkgoWriter, "\n"+message+"\n")
	} else {
		_, err = fmt.Fprintf(ginkgo.GinkgoWriter, "\n"+message+"\n", args...)
	}
	if err != nil {
		panic(err)
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

func DoHTTP(reaction *HTTPReaction, checks *HTTPExpectations) util.DoRequestOsbFunc {
	return func(request *http.Request, client *http.Client) (*http.Response, error) {
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

type HTTPCouple struct {
	Expectations *HTTPExpectations
	Reaction     *HTTPReaction
}

func DoHTTPSequence(sequence []HTTPCouple) util.DoRequestOsbFunc {
	i := 0
	return func(request *http.Request, client *http.Client) (*http.Response, error) {
		r, err := DoHTTP(sequence[i].Reaction, sequence[i].Expectations)(request, client)
		i++
		return r, err
	}
}
