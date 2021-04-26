package common

import (
	"context"
	"github.com/Peripli/service-manager/constant"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gavv/httpexpect"
	"net/http"
)

func CreateInstance(smClient *SMExpect, async bool, expectedStatusCode int, name, planID string) (*httpexpect.Response, string) {
	var instanceID string
	resp := smClient.POST(web.ServiceInstancesURL).
		WithQuery("async", async).
		WithJSON(Object{
			"name":             name,
			"service_plan_id":  planID,
			"maintenance_info": Object{},
		}).Expect().Status(expectedStatusCode)

	if resp.Raw().StatusCode == http.StatusCreated {
		obj := resp.JSON().Object()

		obj.ContainsKey("id").
			ValueEqual("platform_id", types.SMPlatform)

		instanceID = obj.Value("id").String().Raw()
	}

	return resp, instanceID
}

func GetInstanceObjectByID(ctx *TestContext, instanceID string) (*types.ServiceInstance, error) {
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	var err error
	object, err := ctx.SMRepository.Get(context.TODO(), types.ServiceInstanceType, byID)
	if err != nil {
		return nil, err
	}

	return object.(*types.ServiceInstance), nil
}
func ShareInstance(smClient *SMExpect, async bool, expectedStatusCode int, instanceID string) *httpexpect.Response {
	shareInstanceBody := Object{
		"shared": true,
	}

	resp := smClient.PATCH(web.ServiceInstancesURL+"/"+instanceID).
		WithQuery("async", async).
		WithJSON(shareInstanceBody).
		Expect().
		Status(expectedStatusCode)

	resp.JSON().Object().ValueEqual("shared", true)

	return resp
}
func ShareInstanceOnDB(ctx *TestContext, instanceID string) error {
	instance, err := GetInstanceObjectByID(ctx, instanceID)
	if err != nil {
		return err
	}

	instance.Shared = newTrue()
	if _, err := ctx.SMRepository.Update(context.TODO(), instance, types.LabelChanges{}); err != nil {
		return util.HandleStorageError(err, string(instance.GetType()))
	}
	return nil
}

func newTrue() *bool {
	b := true
	return &b
}

func GetReferencePlanOfExistingPlan(ctx *TestContext, byOperator, servicePlanID string) *types.ServicePlan {
	// Retrieve the reference-plan of the service offering.
	byID := query.ByField(query.EqualsOperator, byOperator, servicePlanID)
	planObject, _ := ctx.SMRepository.Get(context.TODO(), types.ServicePlanType, byID)
	plan := planObject.(*types.ServicePlan)

	byID = query.ByField(query.EqualsOperator, "service_offering_id", plan.ServiceOfferingID)
	byName := query.ByField(query.EqualsOperator, "name", constant.ReferencePlanName)
	referencePlanObject, _ := ctx.SMRepository.Get(context.TODO(), types.ServicePlanType, byID, byName)
	if referencePlanObject == nil {
		return nil
	}
	return referencePlanObject.(*types.ServicePlan)
}

func GetPlanByKey(ctx *TestContext, byKeyName, planCatalogID string) *types.ServicePlan {
	byKey := query.ByField(query.EqualsOperator, byKeyName, planCatalogID)
	planObject, _ := ctx.SMRepository.Get(context.TODO(), types.ServicePlanType, byKey)
	return planObject.(*types.ServicePlan)
}

func CreateReferenceInstance(ctx *TestContext, async bool, expectedStatusCode int, referencedInstanceID, referencePlanID, tenantIdentifier, tenantIDValue string) *httpexpect.Response {
	// epsilontal todo: extract the context from the body request and pass it using a new test-filter, in order to test the ownership.
	requestBody := Object{
		"name":             "reference-instance",
		"service_plan_id":  referencePlanID,
		"maintenance_info": "{}",
		"context": Object{
			tenantIdentifier: tenantIDValue,
		},
		"parameters": map[string]string{
			constant.ReferencedInstanceIDKey: referencedInstanceID,
		},
	}
	resp := ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).
		WithQuery("async", async).
		WithJSON(requestBody).
		Expect().
		Status(expectedStatusCode)

	if resp.Raw().StatusCode == http.StatusCreated {
		obj := resp.JSON().Object()

		obj.ContainsKey("id").
			ValueEqual("platform_id", types.SMPlatform)

		// todo: consider returning the instanceID in order to update the test object
		//instanceID = obj.Value("id").String().Raw()
	}

	return resp
}

func CreateBindingByInstanceID(SM *SMExpect, async string, expectedStatusCode int, instanceID string, bindingName string) *httpexpect.Response {
	resp := SM.POST(web.ServiceBindingsURL).
		WithQuery("async", async).
		WithJSON(Object{
			"name":                bindingName,
			"service_instance_id": instanceID,
		}).
		Expect().
		Status(expectedStatusCode)
	obj := resp.JSON().Object()

	if expectedStatusCode == http.StatusCreated {
		obj.ContainsKey("id")
		// todo: consider returning the bindingID in order to update the test object
		//bindingID = obj.Value("id").String().Raw()
	}

	return resp
}
