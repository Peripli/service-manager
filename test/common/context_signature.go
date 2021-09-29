package common

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
)

var (
	CFContext = `{
			"service_id": "%s",
			"plan_id": "%s",
			"parameters":{},
			"context":{
				"platform":"cloudfoundry",
				"organization_guid":"1113aa0-124e-4af2-1526-6bfacf61b111",
				"space_guid":"aaaa1234-da91-4f12-8ffa-b51d0336aaaa",
				"instance_name":"%s",
				"extra_metadata":{
					"key1":"value1",
					"key2":"value2"
				}
			}
		}`
)

func GetVerifyContextHandlerFunc(publicKeyStr string) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		bytes, err := util.BodyToBytes(r.Body)
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}

		//decode the signature
		signatureBytes := gjson.GetBytes(bytes, "context.signature")
		Expect(signatureBytes.Exists()).To(Equal(true), "context should have a signature field")
		signature, err := base64.StdEncoding.DecodeString(signatureBytes.String())
		Expect(err).ToNot(HaveOccurred())

		//decode the public key
		key, err := base64.StdEncoding.DecodeString(publicKeyStr)
		Expect(err).ToNot(HaveOccurred())
		block, _ := pem.Decode(key)
		Expect(block).ToNot(BeNil())
		publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		Expect(err).ToNot(HaveOccurred())

		//read and hash context
		ctxStr := gjson.GetBytes(bytes, "context").String()
		ctxByte, err := sjson.DeleteBytes([]byte(ctxStr), "signature")
		ctxStr = string(ctxByte)
		hashedCtx := sha256.Sum256([]byte(ctxStr))

		//verify signature
		err = rsa.VerifyPKCS1v15(publicKey.(*rsa.PublicKey), crypto.SHA256, hashedCtx[:], signature)
		Expect(err).ToNot(HaveOccurred())

		responseStatus := http.StatusCreated
		if r.Method == http.MethodGet || r.Method == http.MethodPatch {
			responseStatus = http.StatusOK
		}
		if err := util.WriteJSON(rw, responseStatus, Object{}); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			if _, errWrite := rw.Write([]byte(err.Error())); errWrite != nil {
				Expect(errWrite).ToNot(HaveOccurred())
			}
		}
	}
}

func VerifySignatureNotPersisted(ctx *TestContext, objType types.ObjectType, id string) {
	obj, err := ctx.SMRepository.Get(context.TODO(), objType, query.ByField(query.EqualsOperator, "id", id))
	Expect(err).ToNot(HaveOccurred(), "failed to get "+objType.String()+" from db")
	var rawCtx json.RawMessage
	var ctxMap map[string]json.RawMessage
	switch objType {
	case types.ServiceInstanceType:
		rawCtx = obj.(*types.ServiceInstance).Context
	case types.ServiceBindingType:
		rawCtx = obj.(*types.ServiceBinding).Context
	}
	err = json.Unmarshal(rawCtx, &ctxMap)
	Expect(err).ToNot(HaveOccurred())
	Expect(ctxMap).ShouldNot(HaveKey("signature"))
}

func GetOsbProvisionFunc(ctx *TestContext, instanceID, osbURL, catalogServiceID, catalogPlanID string) func() string {
	return func() string {
		ctx.SMWithBasic.PUT(osbURL + "/v2/service_instances/" + instanceID).
			WithJSON(JSONToMap(fmt.Sprintf(CFContext, catalogServiceID, catalogPlanID, "instance-name"))).
			Expect().
			Status(http.StatusCreated)
		return instanceID
	}
}

func GetSMAAPProvisionInstanceFunc(ctx *TestContext, async, planID string) func() string {
	return func() string {
		provisionRequestBody := Object{
			"name":             "test-instance",
			"service_plan_id":  planID,
			"maintenance_info": "{}",
		}
		resp := ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).
			WithQuery("async", async).
			WithJSON(provisionRequestBody).
			Expect().
			Status(http.StatusCreated)

		return resp.JSON().Object().Value("id").String().Raw()
	}
}

func OsbBind(ctx *TestContext, instanceID, bindingID, osbURL, catalogServiceID, catalogPlanID string) *httpexpect.Response {
	return ctx.SMWithBasic.PUT(osbURL + "/v2/service_instances/" + instanceID + "/service_bindings/" + bindingID).
		WithJSON(JSONToMap(fmt.Sprintf(CFContext, catalogServiceID, catalogPlanID, "instance-name"))).
		Expect().
		Status(http.StatusCreated)
}

func SmaapBind(ctx *TestContext, async, instanceID string) string {

	bindingRequestBody := Object{
		"name":                "test-instance-binding",
		"service_instance_id": instanceID,
	}
	resp := ctx.SMWithOAuthForTenant.POST(web.ServiceBindingsURL).
		WithQuery("async", async).
		WithJSON(bindingRequestBody).
		Expect().
		Status(http.StatusCreated)

	return resp.JSON().Object().Value("id").String().Raw()
}

func ProvisionInstanceAndVerifySignature(ctx *TestContext, brokerServer *BrokerServer, provisionFunc func() string, publicKeyStr string) string {
	brokerServer.ServiceInstanceHandler = GetVerifyContextHandlerFunc(publicKeyStr)

	instanceID := provisionFunc()

	VerifySignatureNotPersisted(ctx, types.ServiceInstanceType, instanceID)

	ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/" + instanceID).
		Expect().Status(http.StatusOK).
		JSON().
		Object().Value("context").Object().NotContainsKey("signature")

	return instanceID
}
