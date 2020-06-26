package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	osbc "github.com/kubernetes-sigs/go-open-service-broker-client/v2"
	"github.com/tidwall/sjson"
	"math"
	"net"
	"net/http"
	"net/url"
	"time"
)

type ProvisionResponse struct {
	OrphanMitigation bool
	Async            bool
	DashboardURL     string
	OperationKey     string
	Error            error
}

type BrokerServiceSettings struct {
	OSBClientCreateFunc osbc.CreateFunc
	Repository          storage.Repository
	TenantKey           string
	PollingInterval     time.Duration
}

type BrokerService struct {
	osbClientCreateFunc osbc.CreateFunc
	repository          storage.Repository
	tenantKey           string
	pollingInterval     time.Duration
	context             *ProvisionContext
}

func NewBrokerService(settings BrokerServiceSettings) BrokerService {
	return BrokerService{
		osbClientCreateFunc: settings.OSBClientCreateFunc,
		repository:          settings.Repository,
		tenantKey:           settings.TenantKey,
		pollingInterval:     settings.PollingInterval,
	}
}

type ProvisionContext struct {
	ctx             context.Context
	osbClient       osbc.Client
	serviceBroker   *types.ServiceBroker
	serviceOffering *types.ServiceOffering
	servicePlan     *types.ServicePlan
}

func (sb *BrokerService) ProvisionServiceInstance(instance types.ServiceInstance, ctx context.Context) (ProvisionResponse) {
	var ProvisionServiceInstanceResponse ProvisionResponse;
	instanceContext, err := sb.preparePrerequisites(ctx, &instance)

	if err != nil {
		ProvisionServiceInstanceResponse.Error = fmt.Errorf("failed to prepare provision request: %s", err)
		return ProvisionServiceInstanceResponse
	}

	provisionRequest, err := sb.prepareProvisionRequest(&instance, instanceContext.serviceOffering.CatalogID, instanceContext.servicePlan.CatalogID)

	log.C(ctx).Infof("Sending provision request %s to broker with name %s", logProvisionRequest(provisionRequest), instanceContext.serviceBroker.Name)
	provisionResponse, err := instanceContext.osbClient.ProvisionInstance(provisionRequest)

	if err != nil {
		brokerError := &util.HTTPError{
			ErrorType:   "BrokerError",
			Description: fmt.Sprintf("Failed provisioning request %s: %s", logProvisionRequest(provisionRequest), err),
			StatusCode:  http.StatusBadGateway,
		}
		if shouldStartOrphanMitigation(err) {
			ProvisionServiceInstanceResponse.OrphanMitigation = true
		}
		ProvisionServiceInstanceResponse.Error = brokerError
		return ProvisionServiceInstanceResponse
	}

	if provisionResponse.DashboardURL != nil {
		dashboardURL := *provisionResponse.DashboardURL
		ProvisionServiceInstanceResponse.DashboardURL = dashboardURL
	}

	if provisionResponse.OperationKey != nil {
		ProvisionServiceInstanceResponse.OperationKey = string(*provisionResponse.OperationKey)
	}

	if provisionResponse.Async {
		log.C(ctx).Infof("Successful asynchronous provisioning request %s to broker %s returned response %s",
			logProvisionRequest(provisionRequest), instanceContext.serviceBroker.Name, logProvisionResponse(provisionResponse))
		ProvisionServiceInstanceResponse.Async = true
	} else {
		ProvisionServiceInstanceResponse.Async = false
		log.C(ctx).Infof("Successful synchronous provisioning %s to broker %s returned response %s",
			logProvisionRequest(provisionRequest), instanceContext.serviceBroker.Name, logProvisionResponse(provisionResponse))

	}
	return ProvisionServiceInstanceResponse
}

func (sb *BrokerService) UpdateServiceInstance(instance types.ServiceInstance, ctx context.Context) (ProvisionResponse, error) {
	var provisionServiceInstanceResponse ProvisionResponse;
	requestContext, err := sb.preparePrerequisites(ctx, &instance)

	if err != nil {
		return provisionServiceInstanceResponse, fmt.Errorf("failed to prepare provision request: %s", err)
	}

	instanceObjBeforeUpdate, err := sb.repository.Get(ctx, types.ServiceInstanceType, query.Criterion{
		LeftOp:   "id",
		Operator: query.EqualsOperator,
		RightOp:  []string{instance.ID},
		Type:     query.FieldQuery,
	})
	if err != nil {
		log.C(ctx).WithError(err).Errorf("could not get instance with id '%s'", instance.ID)
		return provisionServiceInstanceResponse, err
	}

	instanceBeforeUpdate := instanceObjBeforeUpdate.(*types.ServiceInstance)

	oldServicePlanObj, err := sb.repository.Get(ctx, types.ServicePlanType, query.Criterion{
		LeftOp:   "id",
		Operator: query.EqualsOperator,
		RightOp:  []string{instanceBeforeUpdate.ServicePlanID},
		Type:     query.FieldQuery,
	})
	if err != nil {
		return provisionServiceInstanceResponse, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: fmt.Sprintf("current service plan with id %s for instance %s no longer exists and instance updates are not allowed", instance.ServicePlanID, instance.Name),
			StatusCode:  http.StatusBadRequest,
		}
	}
	oldServicePlan := oldServicePlanObj.(*types.ServicePlan)

	var updateInstanceResponse *osbc.UpdateInstanceResponse
	updateInstanceRequest, err := sb.prepareUpdateInstanceRequest(&instance, requestContext.serviceOffering.CatalogID, requestContext.servicePlan.CatalogID, oldServicePlan.CatalogID)
	if err != nil {
		return provisionServiceInstanceResponse, fmt.Errorf("faied to prepare update instance request: %s", err)
	}
	log.C(ctx).Infof("Sending update instance request %s to broker with name %s", logUpdateInstanceRequest(updateInstanceRequest), requestContext.serviceBroker.Name)
	updateInstanceResponse, err = requestContext.osbClient.UpdateInstance(updateInstanceRequest)
	if err != nil {
		brokerError := &util.HTTPError{
			ErrorType:   "BrokerError",
			Description: fmt.Sprintf("Failed update instance request %s: %s", logUpdateInstanceRequest(updateInstanceRequest), err),
			StatusCode:  http.StatusBadGateway,
		}
		return provisionServiceInstanceResponse, brokerError
	}

	if updateInstanceResponse.Async {
		provisionServiceInstanceResponse.Async = true
	}

	if len(*updateInstanceResponse.OperationKey) != 0 {
		provisionServiceInstanceResponse.OperationKey = string(*updateInstanceResponse.OperationKey)
	}

	return provisionServiceInstanceResponse, nil
}

//Operation Create - (Service instance  -Failed + OM [delete time set])
func (sb *BrokerService) DeleteServiceInstance(instance types.ServiceInstance, ctx context.Context) (ProvisionResponse, error) {

	var provisionServiceInstanceResponse ProvisionResponse;
	instanceContext, err := sb.preparePrerequisites(ctx, &instance)

	if err != nil {
		return provisionServiceInstanceResponse, fmt.Errorf("failed to prepare provision request: %s", err)
	}

	deprovisionRequest := prepareDeprovisionRequest(&instance, instanceContext.serviceOffering.CatalogID, instanceContext.servicePlan.CatalogID)

	log.C(ctx).Infof("Sending deprovision request %s to broker with name %s", logDeprovisionRequest(deprovisionRequest), instanceContext.serviceBroker.Name)
	deprovisionResponse, err := instanceContext.osbClient.DeprovisionInstance(deprovisionRequest)

	if err != nil {
		if osbc.IsGoneError(err) {
			//log.C(ctx).Infof("Synchronous deprovisioning %s to broker %s returned 410 GONE and is considered success",
			//	logDeprovisionRequest(deprovisionRequest), broker.Name)
			return provisionServiceInstanceResponse, nil
		}
		brokerError := &util.HTTPError{
			ErrorType:   "BrokerError",
			Description: fmt.Sprintf("Failed deprovisioning request %s: %s", logDeprovisionRequest(deprovisionRequest), err),
			StatusCode:  http.StatusBadGateway,
		}

		if shouldStartOrphanMitigation(err) {
			provisionServiceInstanceResponse.OrphanMitigation = true

			//operation.DeletionScheduled = time.Now()
			//operation.Reschedule = false
			//operation.RescheduleTimestamp = time.Time{}
			//if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
			//	return fmt.Errorf("failed to update operation with id %s to schedule orphan mitigation after broker error %s: %s", operation.ID, brokerError, err)
			//}
		}
		return provisionServiceInstanceResponse, brokerError
	}

	if deprovisionResponse.Async {
		log.C(ctx).Infof("Successful asynchronous deprovisioning request %s to broker %s returned response %s",
			logDeprovisionRequest(deprovisionRequest), instanceContext.serviceBroker.Name, logDeprovisionResponse(deprovisionResponse))
		provisionServiceInstanceResponse.Async = true
		//operation.Reschedule = true
		//if operation.RescheduleTimestamp.IsZero() {
		//	operation.RescheduleTimestamp = time.Now()
		//}

		if deprovisionResponse.OperationKey != nil {
			provisionServiceInstanceResponse.OperationKey = string(*deprovisionResponse.OperationKey)
			//operation.ExternalID = string(*deprovisionResponse.OperationKey)
		}
	} else {
		log.C(ctx).Infof("Successful synchronous deprovisioning %s to broker %s returned response %s",
			logDeprovisionRequest(deprovisionRequest), instanceContext.serviceBroker.Name, logDeprovisionResponse(deprovisionResponse))
	}

	return provisionServiceInstanceResponse, nil
}

func prepareDeprovisionRequest(instance *types.ServiceInstance, serviceCatalogID, planCatalogID string) *osbc.DeprovisionRequest {
	return &osbc.DeprovisionRequest{
		InstanceID:        instance.ID,
		AcceptsIncomplete: true,
		ServiceID:         serviceCatalogID,
		PlanID:            planCatalogID,
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}
}

func (sb *BrokerService) UpdateServiceBroker(instance types.ServiceInstance) (bool, error) {
	return true, nil
}

func (sb *BrokerService) DeleteServiceBinding() {

}

func (sb *BrokerService) prepareUpdateInstanceRequest(instance *types.ServiceInstance, serviceCatalogID, planCatalogID, oldCatalogPlanID string) (*osbc.UpdateInstanceRequest, error) {
	instanceContext := make(map[string]interface{})
	if len(instance.Context) != 0 {
		if err := json.Unmarshal(instance.Context, &instanceContext); err != nil {
			return nil, fmt.Errorf("failed to unmarshal already present OSB context: %s", err)
		}
	} else {
		instanceContext = map[string]interface{}{
			"platform":      types.SMPlatform,
			"instance_name": instance.Name,
		}

		if len(sb.tenantKey) != 0 {
			if tenantValue, ok := instance.GetLabels()[sb.tenantKey]; ok {
				instanceContext[sb.tenantKey] = tenantValue[0]
			}
		}

		contextBytes, err := json.Marshal(instanceContext)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal OSB context %+v: %s", instanceContext, err)
		}
		instance.Context = contextBytes
	}

	return &osbc.UpdateInstanceRequest{
		InstanceID:        instance.GetID(),
		AcceptsIncomplete: true,
		ServiceID:         serviceCatalogID,
		PlanID:            &planCatalogID,
		Parameters:        instance.Parameters,
		Context:           instanceContext,
		PreviousValues: &osbc.PreviousValues{
			PlanID: oldCatalogPlanID,
		},
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}, nil
}

func (sb *BrokerService) preparePrerequisites(ctx context.Context, instance *types.ServiceInstance) (*ProvisionContext, error) {
	planObject, err := sb.repository.Get(ctx, types.ServicePlanType, query.ByField(query.EqualsOperator, "id", instance.ServicePlanID))
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)

	serviceObject, err := sb.repository.Get(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "id", plan.ServiceOfferingID))
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceOfferingType.String())
	}
	service := serviceObject.(*types.ServiceOffering)

	brokerObject, err := sb.repository.Get(ctx, types.ServiceBrokerType, query.ByField(query.EqualsOperator, "id", service.BrokerID))
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceBrokerType.String())
	}
	broker := brokerObject.(*types.ServiceBroker)

	tlsConfig, err := broker.GetTLSConfig()
	if err != nil {
		return nil, err
	}

	osbClientConfig := &osbc.ClientConfiguration{
		Name:                broker.Name + " broker client",
		EnableAlphaFeatures: true,
		URL:                 broker.BrokerURL,
		APIVersion:          osbc.LatestAPIVersion(),
	}

	if broker.Credentials.Basic != nil {
		osbClientConfig.AuthConfig = &osbc.AuthConfig{
			BasicAuthConfig: &osbc.BasicAuthConfig{
				Username: broker.Credentials.Basic.Username,
				Password: broker.Credentials.Basic.Password,
			},
		}
	}

	if tlsConfig != nil {
		osbClientConfig.TLSConfig = tlsConfig
	}

	osbClient, err := sb.osbClientCreateFunc(osbClientConfig)
	if err != nil {
		return nil, err
	}

	sb.context = &ProvisionContext{
		osbClient:       osbClient,
		serviceBroker:   broker,
		serviceOffering: service,
		servicePlan:     plan,
	}

	return sb.context, nil
}

func (sb *BrokerService) prepareProvisionRequest(instance *types.ServiceInstance, serviceCatalogID, planCatalogID string) (*osbc.ProvisionRequest, error) {
	instanceContext := make(map[string]interface{})
	if len(instance.Context) != 0 {
		var err error
		instance.Context, err = sjson.SetBytes(instance.Context, "instance_name", instance.Name)
		if err != nil {
			return nil, err
		}

		if err = json.Unmarshal(instance.Context, &instanceContext); err != nil {
			return nil, fmt.Errorf("failed to unmarshal already present OSB context: %s", err)
		}
	} else {
		instanceContext = map[string]interface{}{
			"platform":      types.SMPlatform,
			"instance_name": instance.Name,
		}

		if len(sb.tenantKey) != 0 {
			if tenantValue, ok := instance.GetLabels()[sb.tenantKey]; ok {
				instanceContext[sb.tenantKey] = tenantValue[0]
			}
		}

		contextBytes, err := json.Marshal(instanceContext)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal OSB context %+v: %s", instanceContext, err)
		}
		instance.Context = contextBytes
	}

	provisionRequest := &osbc.ProvisionRequest{
		InstanceID:        instance.GetID(),
		AcceptsIncomplete: true,
		ServiceID:         serviceCatalogID,
		PlanID:            planCatalogID,
		OrganizationGUID:  "-",
		SpaceGUID:         "-",
		Parameters:        instance.Parameters,
		Context:           instanceContext,
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}

	return provisionRequest, nil
}

func (sb *BrokerService) PollServiceInstance(instance types.ServiceInstance, ctx context.Context, externalID string, enableOrphanMitigation bool, rescheduleTimestamp time.Time, category types.OperationCategory, syncPoll bool) (bool, error) {
	instanceContext, err := sb.preparePrerequisites(ctx, &instance)
	if err != nil {
		return false, fmt.Errorf("failed to prepare polling request: %s", err)
	}

	planMaxPollingDuration := time.Duration(instanceContext.servicePlan.MaximumPollingDuration) * time.Second
	leftPollingDuration := time.Duration(math.MaxInt64) // Never tick if plan has not specified max_polling_duration

	if planMaxPollingDuration > 0 {
		// MaximumPollingDuration can span multiple reschedules
		leftPollingDuration = planMaxPollingDuration - (time.Since(rescheduleTimestamp))
		if leftPollingDuration <= 0 { // The Maximum Polling Duration elapsed before this polling start
			return false, sb.processMaxPollingDurationElapsed()
		}
	}

	return sb.pollServiceInstance(instance, ctx, externalID, enableOrphanMitigation, rescheduleTimestamp, category, leftPollingDuration)
}

func (sb *BrokerService) pollServiceInstance(instance types.ServiceInstance, ctx context.Context, externalID string, enableOrphanMitigation bool, rescheduleTimestamp time.Time, category types.OperationCategory, leftPollingDuration time.Duration) (bool, error) {
	var key *osbc.OperationKey
	if len(externalID) != 0 {
		opKey := osbc.OperationKey(externalID)
		key = &opKey
	}

	instanceContext, err := sb.preparePrerequisites(ctx, &instance)

	if err != nil {
		return false, fmt.Errorf("failed to prepare polling request: %s", err)
	}
	pollingRequest := &osbc.LastOperationRequest{
		InstanceID:   instance.ID,
		ServiceID:    &instanceContext.serviceOffering.CatalogID,
		PlanID:       &instanceContext.servicePlan.CatalogID,
		OperationKey: key,
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}

	log.C(ctx).Infof("Sending poll last operation request %s for instance with id %s and name %s", logPollInstanceRequest(pollingRequest), instance.ID, instance.Name)

	pollingResponse, err := instanceContext.osbClient.PollLastOperation(pollingRequest)

	if err != nil {
		if osbc.IsGoneError(err) && category == types.DELETE {
			log.C(ctx).Infof("Successfully finished polling operation for instance with id %s and name %s", instance.ID, instance.Name)
			return true, nil
		} else if isUnreachableBroker(err) {
			log.C(ctx).Errorf("Broker temporarily unreachable. Rescheduling polling last operation request %s to for provisioning of instance with id %s and name %s...",
				logPollInstanceRequest(pollingRequest), instance.ID, instance.Name)
		} else {
			return false, &util.HTTPError{
				ErrorType: "BrokerError",
				Description: fmt.Sprintf("Failed poll last operation request %s for instance with id %s and name %s: %s",
					logPollInstanceRequest(pollingRequest), instance.ID, instance.Name, err),
				StatusCode: http.StatusBadGateway,
			}
		}
	}

	switch pollingResponse.State {
	case osbc.StateInProgress:
		log.C(ctx).Infof("Polling of instance still in progress. Rescheduling polling last operation request %s to for provisioning of instance with id %s and name %s...",
			logPollInstanceRequest(pollingRequest), instance.ID, instance.Name)
	case osbc.StateSucceeded:
		log.C(ctx).Infof("Successfully finished polling operation for instance with id %s and name %s", instance.ID, instance.Name)
		return true, nil
	case osbc.StateFailed:
		log.C(ctx).Infof("Failed polling operation for instance with id %s and name %s with response %s", instance.ID, instance.Name, logPollInstanceResponse(pollingResponse))
		errDescription := ""
		if pollingResponse.Description != nil {
			errDescription = *pollingResponse.Description
		} else {
			errDescription = "no description provided by broker"
		}
		return false, &util.HTTPError{
			ErrorType:   "BrokerError",
			Description: fmt.Sprintf("failed polling operation for instance with id %s and name %s due to polling last operation error: %s", instance.ID, instance.Name, errDescription),
			StatusCode:  http.StatusBadGateway,
		}
	default:
		log.C(ctx).Errorf("invalid state during poll last operation for instance with id %s and name %s: %s. Continuing polling...", instance.ID, instance.Name, pollingResponse.State)
	}

	return false, nil
}

func logPollInstanceResponse(response *osbc.LastOperationResponse) string {
	return fmt.Sprintf("state: %s, description: %s", response.State, strPtrToStr(response.Description))
}

func strPtrToStr(sPtr *string) string {
	if sPtr == nil {
		return ""
	}

	return *sPtr
}

func logPollInstanceRequest(request *osbc.LastOperationRequest) string {
	return fmt.Sprintf("instanceID: %s, planID: %s, serviceID: %s, operationKey: %s",
		request.InstanceID, strPtrToStr(request.PlanID), strPtrToStr(request.ServiceID), opKeyPtrToStr(request.OperationKey))
}

func opKeyPtrToStr(opKey *osbc.OperationKey) string {
	if opKey == nil {
		return ""
	}

	return string(*opKey)
}

func shouldStartOrphanMitigation(err error) bool {
	if httpError, ok := osbc.IsHTTPError(err); ok {
		statusCode := httpError.StatusCode
		is2XX := statusCode >= 200 && statusCode < 300
		is5XX := statusCode >= 500 && statusCode < 600
		return (is2XX && statusCode != http.StatusOK) ||
			statusCode == http.StatusRequestTimeout ||
			is5XX
	}

	if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
		return true
	}

	return false
}

func isUnreachableBroker(err error) bool {
	if timeOutError, ok := err.(net.Error); ok && timeOutError.Timeout() {
		return true
	}
	httpError, ok := osbc.IsHTTPError(err)
	if !ok {
		return false
	}
	return (httpError.StatusCode == http.StatusServiceUnavailable || httpError.StatusCode == http.StatusNotFound)
}

func (sb *BrokerService) processMaxPollingDurationElapsed() error {
	return &util.HTTPError{
		ErrorType:  "BrokerError",
		StatusCode: http.StatusBadGateway,
	}
}

func logProvisionRequest(request *osbc.ProvisionRequest) string {
	return fmt.Sprintf("context: %+v, instanceID: %s, planID: %s, serviceID: %s, acceptsIncomplete: %t",
		request.Context, request.InstanceID, request.PlanID, request.ServiceID, request.AcceptsIncomplete)
}

func logProvisionResponse(response *osbc.ProvisionResponse) string {
	return fmt.Sprintf("async: %t, dashboardURL: %s, operationKey: %s", response.Async, strPtrToStr(response.DashboardURL), opKeyPtrToStr(response.OperationKey))
}
func logDeprovisionRequest(request *osbc.DeprovisionRequest) string {
	return fmt.Sprintf("instanceID: %s, planID: %s, serviceID: %s, acceptsIncomplete: %t",
		request.InstanceID, request.PlanID, request.ServiceID, request.AcceptsIncomplete)
}

func logDeprovisionResponse(response *osbc.DeprovisionResponse) string {
	return fmt.Sprintf("async: %t, operationKey: %s", response.Async, opKeyPtrToStr(response.OperationKey))
}

func logUpdateInstanceRequest(request *osbc.UpdateInstanceRequest) string {
	servicePlanString := ""
	if request.PlanID != nil {
		servicePlanString = "planID: " + (*request.PlanID)
	}
	return fmt.Sprintf("context: %+v, instanceID: %s, %s, serviceID: %s, acceptsIncomplete: %t",
		request.Context, request.InstanceID, servicePlanString, request.ServiceID, request.AcceptsIncomplete)
}
