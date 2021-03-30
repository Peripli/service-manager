package osb

import (
	"context"
	"encoding/json"
	"errors"
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

type referenceInstancePlugin struct {
	repository       storage.TransactionalRepository
	tenantIdentifier string
}

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
	servicePlanID := gjson.GetBytes(req.Body, "service_plan_id").Str
	isReferencePlan, err := p.isReferencePlan(ctx, servicePlanID)
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

	referencedKey := "referenced_instance_id"
	parameters := gjson.GetBytes(req.Body, "parameters").Map()
	referencedInstanceID, exists := parameters[referencedKey]
	// todo: should we validate that the input is string? can be any object for example...
	if !exists {
		return nil, errors.New("missing referenced_instance_id")
	}
	_, err = p.isReferencedShared(ctx, referencedInstanceID.Str)
	if err != nil {
		return nil, err
	}
	// todo: should we handle 201 status for async requests?
	return &web.Response{
		Body:       nil,
		StatusCode: http.StatusCreated,
		Header:     http.Header{},
	}, nil
}

// Deprovision intercepts deprovision requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) Deprovision(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	servicePlanID := gjson.GetBytes(req.Body, "service_plan_id").Str
	isReferencePlan, err := p.isReferencePlan(ctx, servicePlanID)
	if err != nil {
		return nil, err
	}
	if !isReferencePlan {
		return next.Handle(req)
	}

	return &web.Response{
		Body:       nil,
		StatusCode: http.StatusOK,
		Header:     http.Header{},
	}, nil
}

// UpdateService intercepts update service instance requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	// we don't want to allow plan_id and/or parameters changes on a reference service instance
	ctx := req.Context()
	resourceID := req.PathParams["resource_id"]
	if resourceID == "" {
		return nil, errors.New("missing resource ID")
	}

	dbInstanceObject, err := p.getObjectByID(ctx, types.ServiceInstanceType, resourceID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := dbInstanceObject.(*types.ServiceInstance)

	isReferencePlan, err := p.isReferencePlan(ctx, instance.ServicePlanID)
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

	marshal, _ := json.Marshal(nil)
	return &web.Response{
		Body:       marshal,
		StatusCode: http.StatusOK,
		Header:     http.Header{},
	}, nil
}

// PollInstance intercepts poll instance operation requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) PollInstance(req *web.Request, next web.Handler) (*web.Response, error) {
	// todo: no need to support poll instance as we always return sync responses
	return p.assertPlatformID(req, next)
}

// Bind intercepts bind requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	user, _ := web.UserFromContext(ctx)
	platform := &types.Platform{}
	if err := user.Data(platform); err != nil {
		return nil, err
	}
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
		object, err = p.repository.Get(ctx, types.ServiceInstanceType, byID)
		if err != nil {
			if err == util.ErrNotFoundInStorage {
				return next.Handle(req)
			}
			return nil, util.HandleStorageError(err, string(types.ServiceInstanceType))
		}

		instance = object.(*types.ServiceInstance)
		req.Request = req.WithContext(types.ContextWithInstance(req.Context(), instance))

		return p.assertPlatformID(req, next)
	}
	return next.Handle(req)
}

// Unbind intercepts unbind requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) Unbind(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// PollBinding intercepts poll binding operation requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) PollBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// FetchService intercepts get service instance requests and check if the instance owner is the same as the one requesting the operation
func (p *referenceInstancePlugin) FetchService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// FetchBinding intercepts get service binding requests and check if the instance owner is the same as the one requesting the operation
func (p *referenceInstancePlugin) FetchBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

func (p *referenceInstancePlugin) assertPlatformID(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	user, _ := web.UserFromContext(ctx)
	platform := &types.Platform{}
	if err := user.Data(platform); err != nil {
		return nil, err
	}
	if err := platform.Validate(); err != nil {
		log.C(ctx).WithError(err).Errorf("Invalid platform found in context")
		return nil, err
	}

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
	req.Request = req.WithContext(types.ContextWithInstance(req.Context(), instance))

	if platform.ID != instance.PlatformID {
		log.C(ctx).Errorf("Instance with id %s and platform id %s does not belong to platform with id %s", instance.ID, instance.PlatformID, platform.ID)
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "could not find such service instance",
			StatusCode:  http.StatusNotFound,
		}
	}

	return next.Handle(req)
}

func (p *referenceInstancePlugin) validateOwnership(req *web.Request) error {
	ctx := req.Context()
	callerTenantID := gjson.GetBytes(req.Body, "context."+p.tenantIdentifier).String()
	referencedInstanceID := gjson.GetBytes(req.Body, "parameters.referenced_instance_id").String()
	byID := query.ByField(query.EqualsOperator, "id", referencedInstanceID)
	dbReferencedObject, err := p.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := dbReferencedObject.(*types.ServiceInstance)
	referencedOwnerTenantID := instance.Labels["tenant"][0]

	if referencedOwnerTenantID != callerTenantID {
		log.C(ctx).Errorf("Instance owner %s is not the same as the caller %s", referencedOwnerTenantID, callerTenantID)
		return errors.New("could not find such service instance")
	}
	return nil
}

func (p *referenceInstancePlugin) isReferencePlan(ctx context.Context, servicePlanID string) (bool, error) {
	dbPlanObject, err := p.getObjectByID(ctx, types.ServicePlanType, servicePlanID)
	if err != nil {
		return false, err
	}
	plan := dbPlanObject.(*types.ServicePlan)
	return plan.Name == "reference-plan", nil
}

func (p *referenceInstancePlugin) getObjectByID(ctx context.Context, objectType types.ObjectType, resourceID string) (types.Object, error) {
	byID := query.ByField(query.EqualsOperator, "id", resourceID)
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

	if *referencedInstance.Shared != true {
		return false, errors.New("referenced referencedInstance is not shared")
	}
	return true, nil
}

func (p *referenceInstancePlugin) isValidPatchRequest(req *web.Request, instance *types.ServiceInstance) (bool, error) {
	// todo: How can we update labels and do we want to allow the change?
	newPlanID := gjson.GetBytes(req.Body, "service_plan_id").String()
	if instance.ServicePlanID != newPlanID {
		return false, errors.New("can't modify reference's plan")
	}

	parametersRaw := gjson.GetBytes(req.Body, "parameters").Raw
	if parametersRaw != "" {
		return false, errors.New("can't modify reference's parameters")
	}

	return true, nil
}
