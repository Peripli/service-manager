package osb

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/constant"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"net/http"
)

const ReferenceInstancePluginName = "ReferenceInstancePlugin"
const planIDProperty = "plan_id"
const catalogIDProperty = "catalog_id"

type referenceInstancePlugin struct {
	repository       storage.TransactionalRepository
	tenantIdentifier string
}

type osbObject = map[string]interface{}

// NewCheckPlatformIDPlugin creates new plugin that checks the platform_id of the instance
func NewReferenceInstancePlugin(repository storage.TransactionalRepository, tenantIdentifier string) *referenceInstancePlugin {
	return &referenceInstancePlugin{
		repository:       repository,
		tenantIdentifier: tenantIdentifier,
	}
}

// Name returns the name of the plugin
func (p *referenceInstancePlugin) Name() string {
	return ReferenceInstancePluginName
}

func (p *referenceInstancePlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	servicePlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	isReferencePlan, err := p.isReferencePlan(ctx, catalogIDProperty, servicePlanID)
	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	// Ownership validation
	callerTenantID := gjson.GetBytes(req.Body, "context."+p.tenantIdentifier).String()
	if callerTenantID != "" {
		err = p.validateOwnership(req)
		if err != nil {
			return nil, err
		}
	}
	parameters := gjson.GetBytes(req.Body, "parameters").Map()
	referencedInstanceID, exists := parameters[constant.ReferencedInstanceIDKey]
	if !exists {
		return nil, util.HandleInstanceSharingError(util.ErrMissingReferenceParameter, constant.ReferencedInstanceIDKey)
	}
	_, err = p.isReferencedShared(ctx, referencedInstanceID.String())
	if err != nil {
		return nil, err
	}
	return p.generateOSBResponse(ctx, "Provision", nil)
}

// Deprovision intercepts deprovision requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) Deprovision(req *web.Request, next web.Handler) (*web.Response, error) {
	instanceID := req.PathParams["instance_id"]
	if instanceID == "" {
		return next.Handle(req)
	}
	ctx := req.Context()

	dbInstanceObject, err := p.getObjectByOperator(ctx, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	isReferencePlan, err := p.isReferencePlan(ctx, "id", instance.ServicePlanID)

	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	return p.generateOSBResponse(ctx, "Deprovision", nil)
}

// UpdateService intercepts update service instance requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	// we don't want to allow plan_id and/or parameters changes on a reference service instance
	resourceID := req.PathParams["resource_id"]
	if resourceID == "" {
		return next.Handle(req)
	}
	ctx := req.Context()

	dbInstanceObject, err := p.getObjectByOperator(ctx, types.ServiceInstanceType, "id", resourceID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	isReferencePlan, err := p.isReferencePlan(ctx, planIDProperty, instance.ServicePlanID)
	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	_, err = p.isValidPatchRequest(req, instance)
	if err != nil {
		return nil, err
	}

	return p.generateOSBResponse(ctx, "UpdateService", nil)
}

// Bind intercepts bind requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.handleBinding(req, next)
}

// Unbind intercepts unbind requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) Unbind(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.handleBinding(req, next)
}

// FetchBinding intercepts get service binding requests and check if the instance owner is the same as the one requesting the operation
func (p *referenceInstancePlugin) FetchBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.handleBinding(req, next)
}

func (p *referenceInstancePlugin) FetchService(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	instanceID := req.PathParams["instance_id"]

	dbInstanceObject, err := p.getObjectByOperator(ctx, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	isReferencePlan, err := p.isReferencePlan(ctx, "id", instance.ServicePlanID)

	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	return p.generateOSBResponse(ctx, "FetchService", instance)
}

func (p *referenceInstancePlugin) generateOSBResponse(ctx context.Context, method string, instance *types.ServiceInstance) (*web.Response, error) {
	var marshal []byte
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	switch method {
	case "Provision":
		return &web.Response{
			Body:       []byte(`{}`),
			StatusCode: http.StatusCreated,
			Header:     headers,
		}, nil
	case "Deprovision":
		return &web.Response{
			Body:       []byte(`{}`),
			StatusCode: http.StatusOK,
			Header:     headers,
		}, nil
	case "UpdateService":
		return &web.Response{
			Body:       []byte(`{}`),
			StatusCode: http.StatusOK,
			Header:     headers,
		}, nil
	case "FetchService":
		osbResponse, err := p.buildOSBFetchServiceResponse(ctx, instance)
		if err != nil {
			return nil, err
		}
		marshal, err = json.Marshal(osbResponse)
		if err != nil {
			return nil, err
		}
		return &web.Response{
			Body:       marshal,
			StatusCode: http.StatusOK,
			Header:     headers,
		}, nil
	}
	return nil, util.HandleInstanceSharingError(util.ErrUnknownOSBMethod, method)

}

func (p *referenceInstancePlugin) buildOSBFetchServiceResponse(ctx context.Context, instance *types.ServiceInstance) (osbObject, error) {
	serviceOffering, plan, err := p.getServiceOfferingAndPlanByPlanID(ctx, instance.ServicePlanID)
	if err != nil {
		return nil, util.HandleInstanceSharingError(util.ErrNotFoundInStorage, string(types.ServiceOfferingType))
	}
	osbResponse := osbObject{
		"service_id":       serviceOffering.CatalogID,
		"plan_id":          plan.CatalogID,
		"maintenance_info": instance.MaintenanceInfo,
		"parameters": map[string]string{
			constant.ReferencedInstanceIDKey: instance.ReferencedInstanceID,
		},
	}
	return osbResponse, nil
}

// PollBinding intercepts poll binding operation requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) PollBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.handleBinding(req, next)
}

func (p *referenceInstancePlugin) validateOwnership(req *web.Request) error {
	ctx := req.Context()
	callerTenantID := gjson.GetBytes(req.Body, "context."+p.tenantIdentifier).String()
	path := fmt.Sprintf("parameters.%s", constant.ReferencedInstanceIDKey)
	referencedInstanceID := gjson.GetBytes(req.Body, path).String()
	byID := query.ByField(query.EqualsOperator, "id", referencedInstanceID)
	dbReferencedObject, err := p.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := dbReferencedObject.(*types.ServiceInstance)
	referencedOwnerTenantID := instance.Labels["tenant"][0]

	if referencedOwnerTenantID != callerTenantID {
		log.C(ctx).Errorf("Instance owner %s is not the same as the caller %s", referencedOwnerTenantID, callerTenantID)
		return &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "could not find such service instance",
			StatusCode:  http.StatusNotFound,
		}
	}
	return nil
}

func (p *referenceInstancePlugin) isReferencePlan(ctx context.Context, byKey, servicePlanID string) (bool, error) {
	dbPlanObject, err := p.getObjectByOperator(ctx, types.ServicePlanType, byKey, servicePlanID)
	if err != nil {
		return false, err
	}
	plan := dbPlanObject.(*types.ServicePlan)
	return plan.Name == constant.ReferencePlanName, nil
}

func (p *referenceInstancePlugin) getObjectByOperator(ctx context.Context, objectType types.ObjectType, byKey, byValue string) (types.Object, error) {
	byID := query.ByField(query.EqualsOperator, byKey, byValue)
	dbObject, err := p.repository.Get(ctx, objectType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, objectType.String())
	}
	return dbObject, nil
}

func (p *referenceInstancePlugin) isReferencedShared(ctx context.Context, referencedInstanceID string) (bool, error) {
	byID := query.ByField(query.EqualsOperator, "id", referencedInstanceID)
	dbReferencedObject, err := p.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return false, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	referencedInstance := dbReferencedObject.(*types.ServiceInstance)

	if !*referencedInstance.Shared {
		return false, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstanceID)
	}
	return true, nil
}

func (p *referenceInstancePlugin) isValidPatchRequest(req *web.Request, instance *types.ServiceInstance) (bool, error) {
	// epsilontal todo: How can we update labels and do we want to allow the change?
	newPlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	if instance.ServicePlanID != newPlanID {
		return false, util.HandleInstanceSharingError(util.ErrChangingPlanOfReferenceInstance, instance.Name)
	}

	parametersRaw := gjson.GetBytes(req.Body, "parameters").Raw
	if parametersRaw != "" {
		return false, util.HandleInstanceSharingError(util.ErrChangingParametersOfReferenceInstance, instance.Name)
	}

	return true, nil
}

func (p *referenceInstancePlugin) handleBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	instanceID := req.PathParams["instance_id"]
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	object, err := p.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return next.Handle(req)
		}
		return nil, util.HandleStorageError(err, string(types.ServiceInstanceType))
	}

	instance := object.(*types.ServiceInstance)

	if instance.ReferencedInstanceID != "" {
		byID = query.ByField(query.EqualsOperator, "id", instance.ReferencedInstanceID)
		sharedInstanceObject, err := p.repository.Get(ctx, types.ServiceInstanceType, byID)
		if err != nil {
			if err == util.ErrNotFoundInStorage {
				return next.Handle(req)
			}
			return nil, util.HandleStorageError(err, string(types.ServiceInstanceType))
		}

		sharedInstance := sharedInstanceObject.(*types.ServiceInstance)
		req.Request = req.WithContext(types.ContextWithInstance(req.Context(), sharedInstance))
	}
	return next.Handle(req)
}

func (p *referenceInstancePlugin) getServiceOfferingAndPlanByPlanID(ctx context.Context, planID string) (*types.ServiceOffering, *types.ServicePlan, error) {
	plan, err := p.getPlanByID(ctx, planID)
	if err != nil {
		return nil, nil, err
	}
	serviceOffering, err := p.getServiceOfferingByID(ctx, plan.ServiceOfferingID)
	if err != nil {
		return nil, nil, err
	}
	return serviceOffering, plan, nil
}

func (p *referenceInstancePlugin) getPlanByID(ctx context.Context, planID string) (*types.ServicePlan, error) {
	byID := query.ByField(query.EqualsOperator, "id", planID)
	dbPlanObject, err := p.repository.Get(ctx, types.ServicePlanType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := dbPlanObject.(*types.ServicePlan)
	return plan, nil
}

func (p *referenceInstancePlugin) getServiceOfferingByID(ctx context.Context, serviceOfferingID string) (*types.ServiceOffering, error) {
	byID := query.ByField(query.EqualsOperator, "id", serviceOfferingID)
	dbServiceOfferingObject, err := p.repository.Get(ctx, types.ServiceOfferingType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceOfferingType.String())
	}
	serviceOffering := dbServiceOfferingObject.(*types.ServiceOffering)
	return serviceOffering, nil
}
