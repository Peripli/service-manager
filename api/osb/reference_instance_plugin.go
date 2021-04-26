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
const contextKey = "context"

type OperationCategory string

const (
	// Provision represents an operation type for creating a resource
	Provision OperationCategory = "provision"

	// Deprovision represents an operation type for deleting a resource
	Deprovision OperationCategory = "deprovision"

	// UpdateService represents an operation type for updating a resource
	UpdateService OperationCategory = "updateservice"

	// FetchService represents an operation type for updating a resource
	FetchService OperationCategory = "fetchservice"
)

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
func (referencePlugin *referenceInstancePlugin) Name() string {
	return ReferenceInstancePluginName
}

func (referencePlugin *referenceInstancePlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	servicePlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	isReferencePlan, err := referencePlugin.isReferencePlan(ctx, catalogIDProperty, servicePlanID)
	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	// Ownership validation
	path := fmt.Sprintf("context.%s", referencePlugin.tenantIdentifier)
	callerTenantID := gjson.GetBytes(req.Body, path).String()
	if callerTenantID != "" {
		err = referencePlugin.validateOwnership(req)
		if err != nil {
			return nil, err
		}
	}
	parameters := gjson.GetBytes(req.Body, "parameters").Map()
	referencedInstanceID, exists := parameters[constant.ReferencedInstanceIDKey]
	if !exists {
		return nil, util.HandleInstanceSharingError(util.ErrMissingReferenceParameter, constant.ReferencedInstanceIDKey)
	}
	_, err = referencePlugin.isReferencedShared(ctx, referencedInstanceID.String())
	if err != nil {
		return nil, err
	}
	return referencePlugin.generateOSBResponse(ctx, Provision, nil)
}

// Deprovision intercepts deprovision requests and check if the instance is in the platform from where the request comes
func (referencePlugin *referenceInstancePlugin) Deprovision(req *web.Request, next web.Handler) (*web.Response, error) {
	instanceID := req.PathParams["instance_id"]
	if instanceID == "" {
		return next.Handle(req)
	}
	ctx := req.Context()

	dbInstanceObject, err := referencePlugin.getObjectByOperator(ctx, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	isReferencePlan, err := referencePlugin.isReferencePlan(ctx, "id", instance.ServicePlanID)

	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	return referencePlugin.generateOSBResponse(ctx, Deprovision, nil)
}

// UpdateService intercepts update service instance requests and check if the instance is in the platform from where the request comes
func (referencePlugin *referenceInstancePlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	// we don't want to allow plan_id and/or parameters changes on a reference service instance
	resourceID := req.PathParams["resource_id"]
	if resourceID == "" {
		return next.Handle(req)
	}
	ctx := req.Context()

	dbInstanceObject, err := referencePlugin.getObjectByOperator(ctx, types.ServiceInstanceType, "id", resourceID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	isReferencePlan, err := referencePlugin.isReferencePlan(ctx, planIDProperty, instance.ServicePlanID)
	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	_, err = referencePlugin.isValidPatchRequest(req, instance)
	if err != nil {
		return nil, err
	}

	return referencePlugin.generateOSBResponse(ctx, UpdateService, nil)
}

// Bind intercepts bind requests and check if the instance is in the platform from where the request comes
func (referencePlugin *referenceInstancePlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return referencePlugin.handleBinding(req, next)
}

// Unbind intercepts unbind requests and check if the instance is in the platform from where the request comes
func (referencePlugin *referenceInstancePlugin) Unbind(req *web.Request, next web.Handler) (*web.Response, error) {
	return referencePlugin.handleBinding(req, next)
}

// FetchBinding intercepts get service binding requests and check if the instance owner is the same as the one requesting the operation
func (referencePlugin *referenceInstancePlugin) FetchBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return referencePlugin.handleBinding(req, next)
}

func (referencePlugin *referenceInstancePlugin) FetchService(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	instanceID := req.PathParams["instance_id"]

	dbInstanceObject, err := referencePlugin.getObjectByOperator(ctx, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	isReferencePlan, err := referencePlugin.isReferencePlan(ctx, "id", instance.ServicePlanID)

	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	return referencePlugin.generateOSBResponse(ctx, FetchService, instance)
}

func (referencePlugin *referenceInstancePlugin) generateOSBResponse(ctx context.Context, method OperationCategory, instance *types.ServiceInstance) (*web.Response, error) {
	var marshal []byte
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	switch method {
	case FetchService:
		osbResponse, err := referencePlugin.buildOSBFetchServiceResponse(ctx, instance)
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
	default:
		return &web.Response{
			Body:       []byte(`{}`),
			StatusCode: http.StatusOK,
			Header:     headers,
		}, nil
	}
}

func (referencePlugin *referenceInstancePlugin) buildOSBFetchServiceResponse(ctx context.Context, instance *types.ServiceInstance) (osbObject, error) {
	serviceOffering, plan, err := referencePlugin.getServiceOfferingAndPlanByPlanID(ctx, instance.ServicePlanID)
	if err != nil {
		return nil, util.HandleInstanceSharingError(util.ErrNotFoundInStorage, types.ServiceOfferingType.String())
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
func (referencePlugin *referenceInstancePlugin) PollBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return referencePlugin.handleBinding(req, next)
}

func (referencePlugin *referenceInstancePlugin) validateOwnership(req *web.Request) error {
	ctx := req.Context()
	tenantPath := fmt.Sprintf("%s.%s", contextKey, referencePlugin.tenantIdentifier)
	callerTenantID := gjson.GetBytes(req.Body, tenantPath).String()
	path := fmt.Sprintf("parameters.%s", constant.ReferencedInstanceIDKey)
	referencedInstanceID := gjson.GetBytes(req.Body, path).String()
	byID := query.ByField(query.EqualsOperator, "id", referencedInstanceID)
	dbReferencedObject, err := referencePlugin.repository.Get(ctx, types.ServiceInstanceType, byID)
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

func (referencePlugin *referenceInstancePlugin) isReferencePlan(ctx context.Context, byKey, servicePlanID string) (bool, error) {
	dbPlanObject, err := referencePlugin.getObjectByOperator(ctx, types.ServicePlanType, byKey, servicePlanID)
	if err != nil {
		return false, err
	}
	plan := dbPlanObject.(*types.ServicePlan)
	return plan.Name == constant.ReferencePlanName, nil
}

func (referencePlugin *referenceInstancePlugin) getObjectByOperator(ctx context.Context, objectType types.ObjectType, byKey, byValue string) (types.Object, error) {
	byID := query.ByField(query.EqualsOperator, byKey, byValue)
	dbObject, err := referencePlugin.repository.Get(ctx, objectType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, objectType.String())
	}
	return dbObject, nil
}

func (referencePlugin *referenceInstancePlugin) isReferencedShared(ctx context.Context, referencedInstanceID string) (bool, error) {
	byID := query.ByField(query.EqualsOperator, "id", referencedInstanceID)
	dbReferencedObject, err := referencePlugin.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return false, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	referencedInstance := dbReferencedObject.(*types.ServiceInstance)

	if !*referencedInstance.Shared {
		return false, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstanceID)
	}
	return true, nil
}

func (referencePlugin *referenceInstancePlugin) isValidPatchRequest(req *web.Request, instance *types.ServiceInstance) (bool, error) {
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

func (referencePlugin *referenceInstancePlugin) handleBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	instanceID := req.PathParams["instance_id"]
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	object, err := referencePlugin.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return next.Handle(req)
		}
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	instance := object.(*types.ServiceInstance)

	if instance.ReferencedInstanceID != "" {
		byID = query.ByField(query.EqualsOperator, "id", instance.ReferencedInstanceID)
		sharedInstanceObject, err := referencePlugin.repository.Get(ctx, types.ServiceInstanceType, byID)
		if err != nil {
			if err == util.ErrNotFoundInStorage {
				return next.Handle(req)
			}
			return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
		}

		sharedInstance := sharedInstanceObject.(*types.ServiceInstance)
		req.Request = req.WithContext(types.ContextWithSharedInstance(req.Context(), sharedInstance))
	}
	return next.Handle(req)
}

func (referencePlugin *referenceInstancePlugin) getServiceOfferingAndPlanByPlanID(ctx context.Context, planID string) (*types.ServiceOffering, *types.ServicePlan, error) {
	plan, err := referencePlugin.getPlanByID(ctx, planID)
	if err != nil {
		return nil, nil, err
	}
	serviceOffering, err := referencePlugin.getServiceOfferingByID(ctx, plan.ServiceOfferingID)
	if err != nil {
		return nil, nil, err
	}
	return serviceOffering, plan, nil
}

func (referencePlugin *referenceInstancePlugin) getPlanByID(ctx context.Context, planID string) (*types.ServicePlan, error) {
	byID := query.ByField(query.EqualsOperator, "id", planID)
	dbPlanObject, err := referencePlugin.repository.Get(ctx, types.ServicePlanType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := dbPlanObject.(*types.ServicePlan)
	return plan, nil
}

func (referencePlugin *referenceInstancePlugin) getServiceOfferingByID(ctx context.Context, serviceOfferingID string) (*types.ServiceOffering, error) {
	byID := query.ByField(query.EqualsOperator, "id", serviceOfferingID)
	dbServiceOfferingObject, err := referencePlugin.repository.Get(ctx, types.ServiceOfferingType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceOfferingType.String())
	}
	serviceOffering := dbServiceOfferingObject.(*types.ServiceOffering)
	return serviceOffering, nil
}
