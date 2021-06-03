package osb

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/api/common/sharing"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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
	shared := gjson.GetBytes(req.Body, "shared").String()
	if shared != "" {
		log.C(ctx).Errorf("Failed to provision, request body should not contain 'shared' property")
		return nil, util.HandleInstanceSharingError(util.ErrInvalidProvisionRequestWithSharedProperty, "")
	}

	servicePlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	key := "catalog_id"
	isReferencePlan, err := storage.IsReferencePlan(req, is.repository, types.ServicePlanType.String(), key, servicePlanID)
	// If plan not found on provisioning or not reference plan, allow sm to handle the process
	if err == util.ErrNotFoundInStorage || !isReferencePlan {
		return next.Handle(req)
	}
	if err != nil {
		log.C(ctx).Errorf("Failed to retrieve plan %s=%s, while provisioning a reference instance: %s", key, servicePlanID, err)
		return nil, err
	}

	referenceInstanceID, err := sharing.ExtractReferencedInstanceID(req.Context(), is.repository, req.Body, is.tenantIdentifier, func() string {
		return gjson.GetBytes(req.Body, fmt.Sprintf("context.%s", is.tenantIdentifier)).String()
	})
	if err != nil {
		log.C(ctx).Errorf("Failed to extract the referenced instance id: %s", err)
		return nil, err
	}

	log.C(ctx).Infof("Reference Instance validation has passed successfully, instanceID: \"%s\"", referenceInstanceID)

	//OSB spec does not require any fields to be returned
	return util.NewJSONResponse(http.StatusCreated, map[string]string{})
}

// Deprovision validates whether we delete a reference or a shared instance and validates the request before deleting the instance.
func (is *instanceSharingPlugin) Deprovision(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	instanceID := req.PathParams["instance_id"]

	dbInstanceObject, err := storage.GetObjectByField(ctx, is.repository, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		return next.Handle(req)
	}

	instance := dbInstanceObject.(*types.ServiceInstance)

	if instance.IsShared() {
		return deprovisionSharedInstance(ctx, is.repository, req, instance, next)
	}

	isReferencePlan, err := storage.IsReferencePlan(req, is.repository, types.ServicePlanType.String(), "id", instance.ServicePlanID)
	if err != nil {
		log.C(ctx).Errorf("failed to deprovision the reference instance with the plan %s, error: %s", instance.ServicePlanID, err)
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
		referencesArray := types.ObjectListIDsToStringArray(referencesList)
		log.C(ctx).Errorf("Failed to deprovision the shared instance: %s due to existing references %s, error: %s", instance.ID, referencesArray, err)
		return nil, util.HandleReferencesError(util.ErrSharedInstanceHasReferences, referencesArray)
	}
	return next.Handle(req)
}

// UpdateService validates whether we update a reference or a shared instance and validates the request before updating the instance.
func (is *instanceSharingPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	// we don't want to allow plan_id and/or parameters changes on a reference service instance
	instanceID := req.PathParams["instance_id"]
	ctx := req.Context()

	dbInstanceObject, err := storage.GetObjectByField(ctx, is.repository, types.ServiceInstanceType, "id", instanceID)
	if err != nil {
		log.C(ctx).Errorf("Failed to retrieve %s with %s=%s from storage: %s", types.ServiceInstanceType, "id", instanceID, err)
		if err == util.ErrNotFoundInStorage {
			return next.Handle(req)
		}
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	if instance.IsShared() {
		return updateSharedInstance(ctx, is.repository, req, instance, next)
	}
	isReferencePlan, err := storage.IsReferencePlan(req, is.repository, types.ServicePlanType.String(), "id", instance.ServicePlanID)
	if err != nil {
		log.C(ctx).Errorf("Failed to validate the plan %s as reference: %s", instance.ServicePlanID, err)
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	err = sharing.IsValidReferenceInstancePatchRequest(req, instance, planIDProperty)
	if err != nil {
		// error handled via the HandleInstanceSharingError util.
		log.C(ctx).Errorf("Failed to validate patch request of the instance %s, error: %s", instance.ID, err)
		return nil, err
	}

	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

func updateSharedInstance(ctx context.Context, repository storage.Repository, req *web.Request, instance *types.ServiceInstance, next web.Handler) (*web.Response, error) {
	err := isValidSharedInstancePatchRequest(ctx, repository, req, instance)
	if err != nil {
		// error handled via the HandleInstanceSharingError util.
		log.C(ctx).Errorf("Failed to validate patch request of the instance %s, error: %s", instance.ID, err)
		return nil, err
	}
	return next.Handle(req)
}

// Bind via the handleBinding function, it switches the instance's context between the reference instance and the shared instance.
func (is *instanceSharingPlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return is.handleBinding(req, next)
}

// Unbind via the handleBinding function, it switches the instance's context between the reference instance and the shared instance.
func (is *instanceSharingPlugin) Unbind(req *web.Request, next web.Handler) (*web.Response, error) {
	return is.handleBinding(req, next)
}

// FetchBinding via the handleBinding function, it switches the instance's context between the reference instance and the shared instance.
func (is *instanceSharingPlugin) FetchBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return is.handleBinding(req, next)
}

// PollBinding via the handleBinding function, it switches the instance's context between the reference instance and the shared instance.
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

	isReferencePlan, err := storage.IsReferencePlan(req, is.repository, types.ServicePlanType.String(), "id", instance.ServicePlanID)
	if err != nil {
		log.C(ctx).Errorf("Could not validate the plan %s as a reference, error: %s", instance.ServicePlanID, err)
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
	// if changing plan of a shared instance, validate the new plan supports instance sharing.
	if instance.IsShared() && plan.CatalogID != newCatalogID {
		dbNewPlanObject, err := storage.GetObjectByField(ctx, repository, types.ServicePlanType, "catalog_id", newCatalogID)
		if err != nil {
			return util.HandleStorageError(err, types.ServicePlanType.String())
		}
		newPlan := dbNewPlanObject.(*types.ServicePlan)
		if !newPlan.SupportsInstanceSharing() {
			return util.HandleInstanceSharingError(util.ErrNewPlanDoesNotSupportInstanceSharing, "")
		}
	}
	return nil
}

func (is *instanceSharingPlugin) handleBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	instanceID := req.PathParams["instance_id"]
	serviceInstanceObj, err := storage.GetObjectByField(ctx, is.repository, types.ServiceInstanceType, "id", instanceID)

	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return next.Handle(req)
		}
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	instance := serviceInstanceObj.(*types.ServiceInstance)

	// if instance is referecnce, switch the context of the request with the original instance context.
	if len(instance.ReferencedInstanceID) > 0 {
		referencedInstanceObject, err := storage.GetObjectByField(ctx, is.repository, types.ServiceInstanceType, "id", instance.ReferencedInstanceID)
		if err != nil {
			return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
		}
		referencedInstance := referencedInstanceObject.(*types.ServiceInstance)
		servicePlanObj, err := storage.GetObjectByField(ctx, is.repository, types.ServicePlanType, "id", referencedInstance.ServicePlanID)
		if err != nil {
			return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
		}
		servicePlan := servicePlanObj.(*types.ServicePlan)
		//switch context
		req.Request = req.WithContext(types.ContextWithSharedInstance(req.Context(), referencedInstance))
		req.Body, err = sjson.SetBytes(req.Body, "plan_id", servicePlan.CatalogID)
		if err != nil {
			return nil, err
		}
		req.Body, err = sjson.SetBytes(req.Body, "context", referencedInstance.Context)
		if err != nil {
			return nil, err
		}
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
