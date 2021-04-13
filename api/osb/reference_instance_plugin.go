package osb

import (
	"context"
	"encoding/json"
	"errors"
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
const referencedKey = "referenced_instance_id"

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
	servicePlanID := gjson.GetBytes(req.Body, planIDProperty).Str
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
	referencedInstanceID, exists := parameters[referencedKey]
	if !exists {
		return nil, &util.HTTPError{
			ErrorType:   "InvalidRequest",
			Description: fmt.Sprintf("missing parameter %s.", referencedKey),
			StatusCode:  http.StatusBadRequest,
		}
	}
	_, err = p.isReferencedShared(ctx, referencedInstanceID.Str)
	if err != nil {
		return nil, err
	}
	// epsilontal todo: should we handle 201 status for async requests?
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	return &web.Response{
		Body:       []byte(`{}`),
		StatusCode: http.StatusCreated,
		Header:     headers,
	}, nil
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

	return &web.Response{
		Body:       nil,
		StatusCode: http.StatusOK,
		Header:     http.Header{},
	}, nil
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

	marshal, _ := json.Marshal(nil)
	return &web.Response{
		Body:       marshal,
		StatusCode: http.StatusOK,
		Header:     http.Header{},
	}, nil
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
	marshal, _ := json.Marshal(instance)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	return &web.Response{
		Body:       marshal,
		StatusCode: http.StatusOK,
		Header:     headers,
	}, nil
}

// PollBinding intercepts poll binding operation requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) PollBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.handleBinding(req, next)
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

	if *referencedInstance.Shared != true {
		return false, errors.New("referenced referencedInstance is not shared")
	}
	return true, nil
}

func (p *referenceInstancePlugin) isValidPatchRequest(req *web.Request, instance *types.ServiceInstance) (bool, error) {
	// epsilontal todo: How can we update labels and do we want to allow the change?
	newPlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	if instance.ServicePlanID != newPlanID {
		return false, &util.HTTPError{
			ErrorType:   "InvalidRequest",
			Description: fmt.Sprintf("can't modify reference's %s.", planIDProperty),
			StatusCode:  http.StatusBadRequest,
		}
	}

	parametersRaw := gjson.GetBytes(req.Body, "parameters").Raw
	if parametersRaw != "" {
		return false, &util.HTTPError{
			ErrorType:   "InvalidRequest",
			Description: "can't modify reference's parameters.",
			StatusCode:  http.StatusBadRequest,
		}
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
		object, err = p.repository.Get(ctx, types.ServiceInstanceType, byID)
		if err != nil {
			if err == util.ErrNotFoundInStorage {
				return next.Handle(req)
			}
			return nil, util.HandleStorageError(err, string(types.ServiceInstanceType))
		}

		instance = object.(*types.ServiceInstance)
		req.Request = req.WithContext(types.ContextWithInstance(req.Context(), instance))
	}
	return next.Handle(req)
}
