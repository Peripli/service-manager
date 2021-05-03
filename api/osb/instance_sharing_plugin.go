package osb

import (
	"context"
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

const InstanceSharingPluginName = "ReferenceInstancePlugin"
const planIDProperty = "plan_id"

type OperationCategory string

type instanceSharingPlugin struct {
	repository       storage.TransactionalRepository
	tenantIdentifier string
}

type osbObject = map[string]interface{}

// NewInstanceSharingPlugin creates new plugin that handles the instance sharing flows on osb
func NewInstanceSharingPlugin(repository storage.TransactionalRepository, tenantIdentifier string) *instanceSharingPlugin {
	return &instanceSharingPlugin{
		repository:       repository,
		tenantIdentifier: tenantIdentifier,
	}
}

// Name returns the name of the plugin
func (is *instanceSharingPlugin) Name() string {
	return InstanceSharingPluginName
}

func (is *instanceSharingPlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	servicePlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	isReferencePlan, err := storage.IsReferencePlan(ctx, is.repository, types.ServicePlanType.String(), "catalog_id", servicePlanID)
	// If plan not found on provisioning or not reference plan, allow sm to handle the process
	if err == util.ErrNotFoundInStorage || !isReferencePlan {
		return next.Handle(req)
	}
	if err != nil {
		return nil, err
	}

	// Ownership validation
	if is.tenantIdentifier != "" {
		tenantPath := fmt.Sprintf("context.%s", is.tenantIdentifier)
		callerTenantID := gjson.GetBytes(req.Body, tenantPath).String()
		err = storage.ValidateOwnership(is.repository, is.tenantIdentifier, req, callerTenantID)
		if err != nil {
			return nil, err // err returned from validate ownership is "not found" type
		}
	}
	parameters := gjson.GetBytes(req.Body, "parameters").Map()
	referencedInstanceID, exists := parameters[instance_sharing.ReferencedInstanceIDKey]
	if !exists {
		return nil, util.HandleInstanceSharingError(util.ErrMissingReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}
	_, err = storage.IsReferencedShared(ctx, is.repository, referencedInstanceID.String())
	if err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusCreated, map[string]string{})
}

// Deprovision intercepts deprovision requests and check if the instance is in the platform from where the request comes
func (is *instanceSharingPlugin) Deprovision(req *web.Request, next web.Handler) (*web.Response, error) {
	instanceID := req.PathParams["instance_id"]
	ctx := req.Context()

	dbInstanceObject, err := storage.GetObjectByField(ctx, is.repository, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)
	if instance.IsShared() {
		return deprovisionSharedInstance(ctx, is.repository, req, instance, next)
	}
	isReferencePlan, err := storage.IsReferencePlan(ctx, is.repository, types.ServicePlanType.String(), "id", instance.ServicePlanID)

	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

// when deprovisioning a shared instance, validate the instance does not have any references.
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
func (is *instanceSharingPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	// we don't want to allow plan_id and/or parameters changes on a reference service instance
	instanceID := req.PathParams["instance_id"]
	ctx := req.Context()

	dbInstanceObject, err := storage.GetObjectByField(ctx, is.repository, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return next.Handle(req)
		}
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	if instance.IsShared() {
		return updateSharedInstance(ctx, is.repository, req, instance, next)
	}
	isReferencePlan, err := storage.IsReferencePlan(ctx, is.repository, types.ServicePlanType.String(), "id", instance.ServicePlanID)
	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	err = storage.IsValidReferenceInstancePatchRequest(req, instance, planIDProperty)
	if err != nil {
		return nil, err // error handled via the HandleInstanceSharingError util.
	}

	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

func updateSharedInstance(ctx context.Context, repository storage.Repository, req *web.Request, instance *types.ServiceInstance, next web.Handler) (*web.Response, error) {
	err := isValidSharedInstancePatchRequest(ctx, repository, req, instance)
	if err != nil {
		return nil, err // error handled via the HandleInstanceSharingError util.
	}
	return next.Handle(req)
}

// Bind intercepts bind requests and check if the instance is in the platform from where the request comes
func (is *instanceSharingPlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return is.handleBinding(req, next)
}

// Unbind intercepts unbind requests and check if the instance is in the platform from where the request comes
func (is *instanceSharingPlugin) Unbind(req *web.Request, next web.Handler) (*web.Response, error) {
	return is.handleBinding(req, next)
}

// FetchBinding intercepts get service binding requests and check if the instance owner is the same as the one requesting the operation
func (is *instanceSharingPlugin) FetchBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return is.handleBinding(req, next)
}

// PollBinding intercepts poll binding operation requests and check if the instance is in the platform from where the request comes
func (is *instanceSharingPlugin) PollBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return is.handleBinding(req, next)
}

func (is *instanceSharingPlugin) FetchService(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	instanceID := req.PathParams["instance_id"]
	dbInstanceObject, err := storage.GetObjectByField(ctx, is.repository, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	isReferencePlan, err := storage.IsReferencePlan(ctx, is.repository, types.ServicePlanType.String(), "id", instance.ServicePlanID)

	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	body, err := is.buildOSBFetchServiceResponse(ctx, instance)
	if err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusOK, body)
}

func (is *instanceSharingPlugin) buildOSBFetchServiceResponse(ctx context.Context, instance *types.ServiceInstance) (osbObject, error) {
	serviceOffering, plan, err := is.getServiceOfferingAndPlanByPlanID(ctx, instance.ServicePlanID)
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

func isValidSharedInstancePatchRequest(ctx context.Context, repository storage.Repository, req *web.Request, instance *types.ServiceInstance) error {
	newCatalogID := gjson.GetBytes(req.Body, planIDProperty).String()
	dbPlanObject, err := storage.GetObjectByField(ctx, repository, types.ServicePlanType, "id", instance.ServicePlanID)
	if err != nil {
		return err
	}
	plan := dbPlanObject.(*types.ServicePlan)
	if plan.CatalogID != newCatalogID {
		return util.HandleInstanceSharingError(util.ErrChangingPlanOfSharedInstance, instance.ID)
	}
	return nil
}

func (is *instanceSharingPlugin) handleBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	instanceID := req.PathParams["instance_id"]
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	object, err := is.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err == util.ErrNotFoundInStorage {
		return next.Handle(req)
	}
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	instance := object.(*types.ServiceInstance)

	// if instance is referecnce, switch the context of the request with the original instance context.
	if instance.ReferencedInstanceID != "" {
		byID = query.ByField(query.EqualsOperator, "id", instance.ReferencedInstanceID)
		sharedInstanceObject, err := is.repository.Get(ctx, types.ServiceInstanceType, byID)
		if err == util.ErrNotFoundInStorage {
			return next.Handle(req)
		}
		if err != nil {
			return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
		}
		// switch context:
		sharedInstance := sharedInstanceObject.(*types.ServiceInstance)
		req.Request = req.WithContext(types.ContextWithSharedInstance(req.Context(), sharedInstance))
	}
	return next.Handle(req)
}

func (is *instanceSharingPlugin) getServiceOfferingAndPlanByPlanID(ctx context.Context, planID string) (*types.ServiceOffering, *types.ServicePlan, error) {
	dbPlanObject, err := storage.GetObjectByField(ctx, is.repository, types.ServicePlanType, "id", planID)
	if err != nil {
		return nil, nil, err
	}
	plan := dbPlanObject.(*types.ServicePlan)

	dbServiceOfferingObject, err := storage.GetObjectByField(ctx, is.repository, types.ServiceOfferingType, "id", plan.ServiceOfferingID)
	if err != nil {
		return nil, nil, err
	}
	serviceOffering := dbServiceOfferingObject.(*types.ServiceOffering)

	return serviceOffering, plan, nil
}
