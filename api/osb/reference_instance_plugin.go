package osb

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
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
	isReferencePlan, err := storage.IsReferencePlan(ctx, referencePlugin.repository, types.ServicePlanType.String(), "catalog_id", servicePlanID)
	// If plan not found on provisioning, allow sm to handle the issue
	if err == util.ErrNotFoundInStorage {
		return next.Handle(req)
	}
	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	// Ownership validation
	if referencePlugin.tenantIdentifier != "" {
		tenantPath := fmt.Sprintf("context.%s", referencePlugin.tenantIdentifier)
		callerTenantID := gjson.GetBytes(req.Body, tenantPath).String()
		err = storage.ValidateOwnership(referencePlugin.repository, referencePlugin.tenantIdentifier, req, callerTenantID)
		if err != nil {
			return nil, err
		}
	}
	parameters := gjson.GetBytes(req.Body, "parameters").Map()
	referencedInstanceID, exists := parameters[instance_sharing.ReferencedInstanceIDKey]
	if !exists {
		return nil, util.HandleInstanceSharingError(util.ErrMissingReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}
	_, err = storage.IsReferencedShared(ctx, referencePlugin.repository, referencedInstanceID.String())
	if err != nil {
		return nil, err
	}
	return referencePlugin.generateOSBResponse(ctx, Provision, nil)
}

// Deprovision intercepts deprovision requests and check if the instance is in the platform from where the request comes
func (referencePlugin *referenceInstancePlugin) Deprovision(req *web.Request, next web.Handler) (*web.Response, error) {
	instanceID := req.PathParams["instance_id"]
	ctx := req.Context()

	dbInstanceObject, err := storage.GetObjectByField(ctx, referencePlugin.repository, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)
	if instance.Shared != nil && *instance.Shared {
		return deprovisionSharedInstance(ctx, referencePlugin.repository, req, instance, next)
	}
	isReferencePlan, err := storage.IsReferencePlan(ctx, referencePlugin.repository, types.ServicePlanType.String(), "id", instance.ServicePlanID)

	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	return referencePlugin.generateOSBResponse(ctx, Deprovision, nil)
}

func deprovisionSharedInstance(ctx context.Context, repository storage.TransactionalRepository, req *web.Request, instance *types.ServiceInstance, next web.Handler) (*web.Response, error) {
	referencesList, err := storage.GetInstanceReferencesByID(ctx, repository, instance.ID)
	if err != nil {
		log.C(ctx).Errorf("Could not retrieve references of the service instance (%s)s: %v", instance.ID, err)
	}
	if referencesList != nil && referencesList.Len() > 0 {
		return nil, util.HandleReferencesError(util.ErrSharedInstanceHasReferences, types.ObjectListIDsToStringArray(referencesList))
	}
	return next.Handle(req)
}

// UpdateService intercepts update service instance requests and check if the instance is in the platform from where the request comes
func (referencePlugin *referenceInstancePlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	// we don't want to allow plan_id and/or parameters changes on a reference service instance
	instanceID := req.PathParams["instance_id"]
	ctx := req.Context()

	dbInstanceObject, err := storage.GetObjectByField(ctx, referencePlugin.repository, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return next.Handle(req)
		}
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	if instance.Shared != nil && *instance.Shared {
		return updateSharedInstance(ctx, referencePlugin.repository, req, instance, next)
	}
	isReferencePlan, err := storage.IsReferencePlan(ctx, referencePlugin.repository, types.ServicePlanType.String(), "id", instance.ServicePlanID)
	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	err = isValidReferenceInstancePatchRequest(req, instance)
	if err != nil {
		return nil, err
	}

	return referencePlugin.generateOSBResponse(ctx, UpdateService, nil)
}

func updateSharedInstance(ctx context.Context, repository storage.Repository, req *web.Request, instance *types.ServiceInstance, next web.Handler) (*web.Response, error) {
	err := isValidSharedInstancePatchRequest(ctx, repository, req, instance)
	if err != nil {
		return nil, err
	}
	return next.Handle(req)
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
	dbInstanceObject, err := storage.GetObjectByField(ctx, referencePlugin.repository, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	isReferencePlan, err := storage.IsReferencePlan(ctx, referencePlugin.repository, types.ServicePlanType.String(), "id", instance.ServicePlanID)

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
	case Provision:
		return &web.Response{
			Body:       []byte(`{}`),
			StatusCode: http.StatusCreated,
			Header:     headers,
		}, nil
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
			instance_sharing.ReferencedInstanceIDKey: instance.ReferencedInstanceID,
		},
	}
	return osbResponse, nil
}

// PollBinding intercepts poll binding operation requests and check if the instance is in the platform from where the request comes
func (referencePlugin *referenceInstancePlugin) PollBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return referencePlugin.handleBinding(req, next)
}

func isValidReferenceInstancePatchRequest(req *web.Request, instance *types.ServiceInstance) error {
	// epsilontal todo: How can we update labels and do we want to allow the change?
	newPlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	if instance.ServicePlanID != newPlanID {
		return util.HandleInstanceSharingError(util.ErrChangingPlanOfReferenceInstance, instance.ID)
	}

	parametersRaw := gjson.GetBytes(req.Body, "parameters").Raw
	if parametersRaw != "" {
		return util.HandleInstanceSharingError(util.ErrChangingParametersOfReferenceInstance, instance.ID)
	}

	return nil
}
func isValidSharedInstancePatchRequest(ctx context.Context, repository storage.Repository, req *web.Request, instance *types.ServiceInstance) error {
	// epsilontal todo: How can we update labels and do we want to allow the change?
	newPlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	dbPlanObject, err := storage.GetObjectByField(ctx, repository, types.ServicePlanType, "id", instance.ServicePlanID)
	if err != nil {
		return err
	}
	plan := dbPlanObject.(*types.ServicePlan)
	if plan.CatalogID != newPlanID {
		return util.HandleInstanceSharingError(util.ErrChangingPlanOfSharedInstance, instance.ID)
	}
	return nil
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
	dbPlanObject, err := storage.GetObjectByField(ctx, referencePlugin.repository, types.ServicePlanType, "id", planID)
	if err != nil {
		return nil, nil, err
	}
	plan := dbPlanObject.(*types.ServicePlan)

	dbServiceOfferingObject, err := storage.GetObjectByField(ctx, referencePlugin.repository, types.ServiceOfferingType, "id", plan.ServiceOfferingID)
	if err != nil {
		return nil, nil, err
	}
	serviceOffering := dbServiceOfferingObject.(*types.ServiceOffering)

	return serviceOffering, plan, nil
}
