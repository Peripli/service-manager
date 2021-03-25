package osb

import (
	"context"
	"encoding/json"
	"github.com/tidwall/gjson"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const ReferenceInstancePluginName = "ReferenceInstancePlugin"

type referenceInstancePlugin struct {
	repository storage.TransactionalRepository
}

// NewCheckPlatformIDPlugin creates new plugin that checks the platform_id of the instance
func NewReferenceInstancePlugin(repository storage.TransactionalRepository) *referenceInstancePlugin {
	return &referenceInstancePlugin{
		repository: repository,
	}
}

// Name returns the name of the plugin
func (p *referenceInstancePlugin) Name() string {
	return ReferenceInstancePluginName
}

func (p *referenceInstancePlugin) generateReferenceInstance(body []byte, instanceID, planID string) *types.ServiceInstance {

	//UUID, _ := uuid.NewV4()
	currentTime := time.Now().UTC()

	referencedInstanceID := gjson.GetBytes(body, "parameters.referenced_instance_id").String()
	if referencedInstanceID == "" {
		referencedInstanceID = gjson.GetBytes(body, "referenced_instance_id").String()
	}

	instance := &types.ServiceInstance{
		Base: types.Base{
			//ID:        UUID.String(),
			ID:        instanceID,
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
			Labels: types.Labels{
				"tenant": []string{"tenant_value"},
			},
			Ready: true,
		},
		Name:                 gjson.GetBytes(body, "name").String(),
		ServicePlanID:        planID,
		PlatformID:           gjson.GetBytes(body, "context.platform").String(),
		MaintenanceInfo:      json.RawMessage(gjson.GetBytes(body, "maintenance_info").String()),
		Context:              json.RawMessage(gjson.GetBytes(body, "contextt").String()),
		Usable:               true,
		ReferencedInstanceID: referencedInstanceID,
	}

	return instance
}
func (p *referenceInstancePlugin) createReferenceInstance(ctx context.Context, instance *types.ServiceInstance) error {
	sharingErr := p.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		_, err := storage.Create(ctx, instance)
		if err != nil {
			log.C(ctx).Errorf("Could not update shared property for instance (%s): %v", instance.ID, err)
			return err
		}
		return nil
	})
	return sharingErr
}

// Deprovision intercepts deprovision requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
	referencedKey := "referenced_instance_id"
	parameters := gjson.GetBytes(req.Body, "parameters").Map()
	_, exists := parameters[referencedKey]
	if !exists {
		return next.Handle(req)
	}

	//ctx := req.Context()

	//planID := gjson.GetBytes(req.Body, "plan_id").String()

	//byID := query.ByField(query.EqualsOperator, "catalog_id", planID)
	//planObject, err := p.repository.Get(ctx, types.ServicePlanType, byID)
	//if err != nil {
	//	return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	//}
	//plan := planObject.(*types.ServicePlan)
	//if isReferencePlan(plan) {
	//	// set as !isReferencePlan
	//	return nil, errors.New("plan_id is not a reference plan")
	//}

	//byID = query.ByField(query.EqualsOperator, "id", referencedInstanceID.Str)
	//referencedObject, err := p.repository.Get(ctx, types.ServiceInstanceType, byID)
	//if err != nil {
	//	return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	//}
	//instance := referencedObject.(*types.ServiceInstance)

	//if !instance.Shared {
	//	return nil, errors.New("referenced instance is not shared")
	//}
	//instanceRequestBody := decodeRequestToObject(req.Body)

	//instanceID := req.PathParams["instance_id"]
	//generatedReferenceInstance := p.generateReferenceInstance(req.Body, instanceID, plan.ID)
	//err = p.createReferenceInstance(ctx, generatedReferenceInstance)
	//if err != nil {
	//	return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	//}
	return &web.Response{
		Body:       nil,
		StatusCode: http.StatusCreated,
		Header:     http.Header{},
	}, nil
	//return util.NewJSONResponse(http.StatusCreated, instance)
}

// Deprovision intercepts deprovision requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) Deprovision(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// UpdateService intercepts update service instance requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// PollInstance intercepts poll instance operation requests and check if the instance is in the platform from where the request comes
func (p *referenceInstancePlugin) PollInstance(req *web.Request, next web.Handler) (*web.Response, error) {
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
