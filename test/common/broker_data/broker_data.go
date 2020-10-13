package broker_data

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	"github.com/tidwall/gjson"

	. "github.com/onsi/gomega"
)

const (
	// NameKey is used for getting/setting service/plan name in json.
	NameKey = "name"

	PlatformName         string = "metadata.supportedPlatformNames"
	ExcludedPlatformName string = "metadata.excludedPlatformNames"
	PlatformType         string = "metadata.supportedPlatforms"
)

// New returns a BrokerData object. isTenantScoped bool parameter specifies whether the broker will be tenant scoped.
func New(isTenantScoped bool) *BrokerData {
	return &BrokerData{
		catalog:            common.NewEmptySBCatalog(),
		plans:    []PlanServicePair{},
	}
}

// PlanServicePair struct identifies a given plan within a given broker.
type PlanServicePair struct {
	PlanName    string
	ServiceName string
}

// BrokerData struct is a test utility for managing service brokers.
type BrokerData struct {
	isTenantScoped bool

	catalog      common.SBCatalog
	labels       common.Object
	labelChanges []common.Object

	plans    []PlanServicePair

	brokerID     string
	brokerServer *common.BrokerServer
	brokerObject common.Object

	planServiceToPlanID map[PlanServicePair]string
	serviceNameToID     map[string]string
}

// CreateBrokerInSM registers a broker with the given data in SM.
func (bd *BrokerData) CreateBrokerInSM(ctx *common.TestContext) {
	var labels common.Object
	if len(bd.labels) != 0 {
		labels = common.Object{
			"labels": bd.labels,
		}
	}
	var expect *common.SMExpect
	if bd.isTenantScoped {
		expect = ctx.SMWithOAuthForTenant
	} else {
		expect = ctx.SMWithOAuth
	}
	bd.brokerID, bd.brokerObject, bd.brokerServer = ctx.RegisterBrokerWithCatalogAndLabelsExpect(bd.catalog, labels, expect).GetBrokerAsParams()
	bd.updatePlanAndServiceIDs(ctx)
}

// IsTenantScoped tells if the broker is tenant scoped or not
func (bd *BrokerData) IsTenantScoped() bool {
	return bd.isTenantScoped
}
func (bd *BrokerData) GetPlansFromStorage(ctx *common.TestContext) *types.ServicePlans {
	plansIDS := bd.GetPlanIDs()
	byPlansID := query.ByField(query.InOperator, "id", plansIDS...)
	plans, err := ctx.SMRepository.List(context.TODO(), types.ServicePlanType, byPlansID)
	Expect(err).ToNot(HaveOccurred())
	return plans.(*types.ServicePlans)
}

// GetInstancesForPlans returns all instances for the given planServicePairs.
// If broker was never registered, nil is returned.
func (bd *BrokerData) GetInstancesForPlans(ctx *common.TestContext, planServicePairs ...PlanServicePair) *types.ServiceInstances {
	if len(bd.brokerID) == 0 {
		return nil
	}
	byPlanIDsQuery := query.ByField(query.InOperator, "service_plan_id", bd.GetPlanIDs(planServicePairs...)...)
	serviceInstances, err := ctx.SMRepository.List(context.TODO(), types.ServiceInstanceType, byPlanIDsQuery)
	if err != nil {
		Expect(fmt.Errorf("could not get service instances: %s", err)).ToNot(HaveOccurred())
	}
	return serviceInstances.(*types.ServiceInstances)
}

// RemoveInstances removes all service instances for given planServicePairs.
// If force flag is true all bindings for these instances will be deleted as well.
// If broker was never registered, nothing will be done.
func (bd *BrokerData) RemoveInstances(ctx *common.TestContext, force bool, planServicePairs ...PlanServicePair) {
	if len(bd.brokerID) == 0 {
		return
	}
	if force {
		bd.RemoveBindings(ctx, planServicePairs...)
	}
	serviceInstances := bd.GetInstancesForPlans(ctx, planServicePairs...)
	for i := 0; i < serviceInstances.Len(); i++ {
		serviceInstance := serviceInstances.ItemAt(i).(*types.ServiceInstance)
		if err := common.DeleteInstance(ctx, serviceInstance.ID, serviceInstance.ServicePlanID); err != nil {
			Expect(fmt.Errorf("could not delete service instance: %s", err)).ToNot(HaveOccurred())
		}
	}
}

// RemoveBindings removes all service bindings for given planServicePairs.
// If broker was never registered, nothing will be done.
func (bd *BrokerData) RemoveBindings(ctx *common.TestContext, planServicePairs ...PlanServicePair) {
	if len(bd.brokerID) == 0 {
		return
	}
	serviceInstances := bd.GetInstancesForPlans(ctx, planServicePairs...)
	serviceInstancesLen := serviceInstances.Len()
	if serviceInstancesLen == 0 {
		return
	}
	serviceInstanceIDs := make([]string, 0, serviceInstancesLen)
	for _, serviceInstance := range serviceInstances.ServiceInstances {
		serviceInstanceIDs = append(serviceInstanceIDs, serviceInstance.ID)
	}
	byServiceInstanceIDQuery := query.ByField(query.InOperator, "service_instance_id", serviceInstanceIDs...)
	serviceBindings, err := ctx.SMRepository.List(context.TODO(), types.ServiceBindingType, byServiceInstanceIDQuery)
	if err != nil {
		Expect(fmt.Errorf("could not get service bindings: %s", err)).ToNot(HaveOccurred())
	}
	for i := 0; i < serviceBindings.Len(); i++ {
		serviceBinding := serviceBindings.ItemAt(i).(*types.ServiceBinding)
		if err = common.DeleteBinding(ctx, serviceBinding.ID, serviceBinding.ServiceInstanceID); err != nil {
			Expect(fmt.Errorf("could not delete service bindings: %s", err)).ToNot(HaveOccurred())
		}
	}
	if err != nil {
		Expect(fmt.Errorf("could not delete service bindings: %s", err)).ToNot(HaveOccurred())
	}
}

// RemoveAllInstances removes all service instances for plans of this broker.
// If force parameter is true - all bindings for these instances will be removed as well.
// If broker was never registered, nothing will be done.
func (bd *BrokerData) RemoveAllInstances(ctx *common.TestContext, force bool) {
	bd.RemoveInstances(ctx, force, bd.GetAllPlans()...)
}

// RemoveAllBindings removes all service bindings for plans of this broker.
// If broker was never registered, nothing will be done.
func (bd *BrokerData) RemoveAllBindings(ctx *common.TestContext) {
	bd.RemoveBindings(ctx, bd.GetAllPlans()...)
}

// RemoveBrokerInSM removes the registered broker in SM. If broker was never registered, nothing will be done.
// The force flag states whether the service instances and bindings for the broker plans should be deleted as well.
func (bd *BrokerData) RemoveBrokerInSM(ctx *common.TestContext, force bool) {
	if len(bd.brokerID) != 0 {
		if force {
			bd.RemoveAllInstances(ctx, force)
		}
		ctx.CleanupBroker(bd.brokerID)
		bd.brokerID = ""
		bd.brokerServer = nil
		bd.brokerObject = nil
		bd.planServiceToPlanID = make(map[PlanServicePair]string)
		bd.serviceNameToID = make(map[string]string)
		bd.updateLabels()
	}
}

// UpdateBrokerInSM updates the registered broker in SM. Panics if broker was never registered.
func (bd *BrokerData) UpdateBrokerInSM(ctx *common.TestContext, resync bool) *httpexpect.Response {
	if bd.brokerServer == nil {
		Expect(errors.New("could not update broker, it is not registered yet")).ToNot(HaveOccurred())
	}
	patchBody := common.Object{}
	if len(bd.labelChanges) != 0 {
		patchBody = common.Object{
			"labels": bd.labelChanges,
		}
	}
	var expect *common.SMExpect
	if bd.isTenantScoped {
		expect = ctx.SMWithOAuthForTenant
	} else {
		expect = ctx.SMWithOAuth
	}
	var resp *httpexpect.Response
	if resync  {
		resp = expect.PATCH(web.ServiceBrokersURL+"/"+bd.brokerID).WithQuery("resync", resync).WithJSON(patchBody).Expect()
	} else {
		resp = expect.PATCH(web.ServiceBrokersURL + "/" + bd.brokerID).WithJSON(patchBody).Expect()
	}
	if resp.Raw().StatusCode == http.StatusOK {
		bd.updatePlanAndServiceIDs(ctx)
		bd.updateLabels()
	}
	return resp
}

// AddServices adds new services from json strings. Labels/label changes are calculated.
func (bd *BrokerData) AddServices(serviceJSONs ...string) (planServicePairsAddedToCatalog []PlanServicePair) {
	for _, serviceJSON := range serviceJSONs {
		bd.catalog.AddService(serviceJSON)

		serviceName := gjson.Get(serviceJSON, NameKey).String()
		servicePlans := gjson.Get(serviceJSON, "plans").Array()

		for _, plan := range servicePlans {
			planName := gjson.Get(plan.Raw, NameKey).String()

			planServicePair := PlanServicePair{
				PlanName:    planName,
				ServiceName: serviceName,
			}
			planServicePairsAddedToCatalog = append(planServicePairsAddedToCatalog, planServicePair)
			bd.addPlan(planServicePair)
		}
	}
	bd.updateBrokerServerCatalog()
	return planServicePairsAddedToCatalog
}

// AddPlans adds new plans from json strings to a specific service.
// Panics if service is not found in the catalog. Labels/label changes are calculated.
func (bd *BrokerData) AddPlans(serviceName string, planJSONs ...string) (planServicePairsAddedToCatalog []PlanServicePair) {
	serviceIndex, _ := bd.GetServiceFromCatalog(serviceName)

	for _, planJSON := range planJSONs {
		bd.catalog.AddPlanToService(planJSON, serviceIndex)

		planName := gjson.Get(planJSON, NameKey).String()

		planServicePair := PlanServicePair{
			PlanName:    planName,
			ServiceName: serviceName,
		}
		planServicePairsAddedToCatalog = append(planServicePairsAddedToCatalog, planServicePair)
		bd.addPlan(planServicePair)
	}
	bd.updateBrokerServerCatalog()
	return planServicePairsAddedToCatalog
}

// RemoveServices removes services by name. Panics if any of the service are not found in the catalog.
func (bd *BrokerData) RemoveServices(serviceNames ...string) {
	for _, serviceName := range serviceNames {
		serviceIndex, _ := bd.GetServiceFromCatalog(serviceName)

		for _, planServicePair := range bd.GetAllPlans() {
			if planServicePair.ServiceName == serviceName {
				bd.RemovePlans(planServicePair)
			}
		}
		bd.catalog.RemoveService(serviceIndex)
	}
	bd.updateBrokerServerCatalog()
}

// RemovePlans removes plans by plan/service pairs. Panics if any of the service/plan pairs are not found in the catalog.
func (bd *BrokerData) RemovePlans(planServicePairs ...PlanServicePair) {
	for _, plan := range planServicePairs {
		for index, planServicePair := range bd.plans {
			if planServicePair == plan {
				bd.plans = append(bd.plans[:index], bd.plans[index+1:]...)
				break
			}
		}
		bd.addLabelChangeForPlanRemoval(plan)

		serviceIndexInCatalog, planIndexInCatalog := bd.getPlanIndexInCatalog(plan)
		if planIndexInCatalog == -1 {
			Expect(
				fmt.Errorf("could not find service %s, plan %s in catalog %s", plan.ServiceName, plan.PlanName, bd.catalog)).
				ToNot(HaveOccurred())
		}
		bd.catalog.RemovePlan(serviceIndexInCatalog, planIndexInCatalog)
	}
	bd.updateBrokerServerCatalog()

}

// GetPlanIDs returns the plan ids in SM. Panics if plan/service pair is not found or broker is not registered/updated.
func (bd *BrokerData) GetPlanIDs(plans ...PlanServicePair) (planIDs []string) {
	Expect(bd.brokerServer).ToNot(BeNil())
	for _, planServicePair := range plans {
		planID := bd.planServiceToPlanID[planServicePair]
		if len(planID) == 0 {
			Expect(
				fmt.Errorf("could not find service %s, plan %s in catalog %s", planServicePair.ServiceName, planServicePair.PlanName, bd.catalog)).
				ToNot(HaveOccurred())
		}
		planIDs = append(planIDs, planID)
	}
	return
}

// GetServiceIDs returns the service ids for a given service names in SM.
// Panics if plan/service pair is not found or broker is not registered/updated.
func (bd *BrokerData) GetServiceIDs(serviceNames ...string) (serviceIDs []string) {
	Expect(bd.brokerServer).ToNot(BeNil())
	for _, serviceName := range serviceNames {
		serviceID := bd.serviceNameToID[serviceName]
		if len(serviceID) == 0 {
			Expect(
				fmt.Errorf("could not find service %s in catalog %s", serviceName, bd.catalog)).
				ToNot(HaveOccurred())
		}
		serviceIDs = append(serviceIDs, serviceID)
	}
	return
}

// GetAllServiceIDs returns all service ids of this broker in SM.
// Panics if plan/service pair is not found or broker is not registered/updated.
func (bd *BrokerData) GetAllServiceIDs() (serviceIDs []string) {
	return bd.GetServiceIDs(bd.GetAllServiceNames()...)
}

// GetAllPlanIDs returns all plan ids in SM for the broker. Panics broker is not registered/updated.
func (bd *BrokerData) GetAllPlanIDs() (planIDs []string) {
	return bd.GetPlanIDs(bd.plans...)
}

// GetAllServiceNames returns all service names that are in the catalog.
func (bd *BrokerData) GetAllServiceNames() (serviceNames []string) {
	services := bd.GetServices()
	serviceNames = make([]string, 0, len(services))
	for serviceName := range services {
		serviceNames = append(serviceNames, serviceName)
	}
	return
}

// GetServices returns a map with all services with their plans. The map key is the service name.
func (bd *BrokerData) GetServices() (services map[string][]PlanServicePair) {
	services = make(map[string][]PlanServicePair)
	for _, planServicePair := range bd.GetAllPlans() {
		services[planServicePair.ServiceName] = append(services[planServicePair.ServiceName], planServicePair)
	}
	return
}

// GetAllPlans returns all commercial plans.
func (bd *BrokerData) GetAllPlans() (planServicePairs []PlanServicePair) {
	return bd.plans
}

// GetPlansForServices returns all plans for a given set of services.
func (bd *BrokerData) GetPlansForServices(serviceNames ...string) (planServicePairs []PlanServicePair) {
	for _, serviceName := range serviceNames {
		planServicePairs = append(planServicePairs, bd.getPlansForService(serviceName, func(planJSON string) bool {
			return true
		})...)
	}
	return
}

// GetRegisteredBrokerID returns the ID of the registered broker or empty string if broker is not registered.
func (bd *BrokerData) GetRegisteredBrokerID() string {
	return bd.brokerID
}

// GetRegisteredBrokerURL returns the URL of the registered broker or empty string if broker is not registered.
func (bd *BrokerData) GetRegisteredBrokerURL() string {
	if bd.brokerServer != nil {
		return bd.brokerServer.URL()
	}
	return ""
}

// GetRegisteredBrokerServer returns the http server of the registered broker or nil if broker is not registered.
func (bd *BrokerData) GetRegisteredBrokerServer() *common.BrokerServer {
	return bd.brokerServer
}

// GetRegisteredBrokerObject returns the Object returned from SM when broker was registered.
// Returns empty object if broker was not registered
func (bd *BrokerData) GetRegisteredBrokerObject() common.Object {
	return bd.brokerObject
}

// GetLabels returns the labels of the currently registered broker.
// If broker is not registered yet, returns the labels as they would be passed when registering the broker.
func (bd *BrokerData) GetLabels() (labels common.Object) {
	labels = common.Object{}
	for k, v := range bd.labels {
		labels[k] = v
	}
	return labels
}

// GetPendingLabelChanges returns the label changes as they would be passed to an update broker request.
// If broker is not registered yet, returns nil.
func (bd *BrokerData) GetPendingLabelChanges() []common.Object {
	return bd.labelChanges
}

// SetLabelChanges sets the label changes that will be used for the next broker update.
func (bd *BrokerData) SetLabelChanges(labelChanges []common.Object) {
	bd.labelChanges = labelChanges
}

// GetCatalog returns the broker catalog json.
func (bd *BrokerData) GetCatalog() string {
	return string(bd.catalog)
}

// GetServiceFromCatalog returns the index of the service in the catalog and the service as json string.
// Panics if service is not found in catalog.
func (bd *BrokerData) GetServiceFromCatalog(serviceName string) (serviceIndexInCatalog int, serviceJSON string) {
	catalogServices := gjson.Get(string(bd.catalog), "services").Array()
	for index, service := range catalogServices {
		serviceJSON := service.Raw
		if gjson.Get(serviceJSON, fmt.Sprintf(NameKey)).String() == serviceName {
			return index, serviceJSON
		}
	}
	Expect(fmt.Errorf("could not find service '%s' in catalog %s", serviceName, bd.catalog)).ToNot(HaveOccurred())
	return
}

// GetPlanFromCatalog returns the index of the service and plan in the catalog and the plan as json string.
// Panics if plan is not found in catalog.
func (bd *BrokerData) GetPlanFromCatalog(planServicePair PlanServicePair) (serviceIndexInCatalog int, planIndexInCatalog int, planJSON string) {
	serviceIndexInCatalog, planIndexInCatalog = bd.getPlanIndexInCatalog(planServicePair)
	if planIndexInCatalog == -1 {
		Expect(
			fmt.Errorf("could not find service '%s', plan '%s' in catalog %s", planServicePair.ServiceName, planServicePair.PlanName, bd.catalog)).
			ToNot(HaveOccurred())
	}
	planJSON = gjson.Get(string(bd.catalog), fmt.Sprintf("services.%d.plans.%d", serviceIndexInCatalog, planIndexInCatalog)).Raw
	return
}

func GetPlanMetadataValuesBySupportedPlatforms(platformMetadataPropertyKey string, supportedPlatforms []*types.Platform) []string {
	result := make([]string, 0)
	for _, supportedPlatform := range supportedPlatforms {
		if platformMetadataPropertyKey == PlatformType {
			result = append(result, supportedPlatform.Type)
		} else {
			result = append(result, supportedPlatform.Name)
		}
	}
	return result
}

func (bd *BrokerData) UpdatePlanSupportedPlatforms(platformMetadataPropertyKey string, serviceIndexInCatalog int, planIndexInCatalog int, supportedPlatforms ...string) {
	catalog := string(bd.catalog)

	var err error
	count := len(gjson.Get(catalog, fmt.Sprintf("services.%d.plans.%d.%s", serviceIndexInCatalog, planIndexInCatalog, platformMetadataPropertyKey)).Array())
	for count > 0 {
		catalog, err = sjson.Delete(catalog, fmt.Sprintf("services.%d.plans.%d.%s.0", serviceIndexInCatalog, planIndexInCatalog, platformMetadataPropertyKey))
		Expect(err).ToNot(HaveOccurred())

		count = len(gjson.Get(catalog, fmt.Sprintf("services.%d.plans.%d.%s", serviceIndexInCatalog, planIndexInCatalog, platformMetadataPropertyKey)).Array())
	}

	for _, platformType := range supportedPlatforms {
		catalog, err = sjson.Set(catalog, fmt.Sprintf("services.%d.plans.%d.%s.-1", serviceIndexInCatalog, planIndexInCatalog, platformMetadataPropertyKey), platformType)
		Expect(err).ToNot(HaveOccurred())
	}

	bd.catalog = common.SBCatalog(catalog)
	bd.updateBrokerServerCatalog()
}

func (bd *BrokerData) getPlanIndexInCatalog(plan PlanServicePair) (serviceIndexInCatalog, planIndexInCatalog int) {
	var serviceJSON string
	planIndexInCatalog = -1
	serviceIndexInCatalog, serviceJSON = bd.GetServiceFromCatalog(plan.ServiceName)

	catalogPlans := gjson.Get(serviceJSON, "plans").Array()
	for index, catalogPlan := range catalogPlans {
		catalogPlanName := gjson.Get(catalogPlan.Raw, fmt.Sprintf(NameKey)).String()
		if catalogPlanName == plan.PlanName {
			planIndexInCatalog = index
			break
		}
	}
	if planIndexInCatalog == -1 {
		return -1, -1
	}
	return
}

func (bd *BrokerData) getPlansForService(serviceName string, planFilterFunc func(planJSON string) bool) (planServicePairs []PlanServicePair) {
	_, serviceJSON := bd.GetServiceFromCatalog(serviceName)
	plans := gjson.Get(serviceJSON, "plans").Array()
	for _, plan := range plans {
		if planFilterFunc(plan.Raw) {
			planServicePairs = append(planServicePairs, PlanServicePair{
				PlanName:    gjson.Get(plan.Raw, NameKey).String(),
				ServiceName: serviceName,
			})
		}
	}
	return
}

func (bd *BrokerData) addLabelChangeForNewPlan(planServicePair PlanServicePair) {
	if bd.isTenantScoped {
		return
	}
	if bd.brokerServer == nil { // broker not registered => updating labels
		bd.updateLabels()
		return
	}
	bd.labelChanges = append(bd.labelChanges, common.Object{
		"op":  "add_values",
		"key": "plan", // TODO: What key to use?
		"values": common.Array{
			planServicePair.ServiceName + ":" + planServicePair.PlanName,
		},
	})
}

func (bd *BrokerData) addLabelChangeForPlanRemoval(planServicePair PlanServicePair) {
	if bd.isTenantScoped {
		return
	}
	if bd.brokerServer == nil { // broker not registered => updating labels
		bd.updateLabels()
		return
	}
	bd.labelChanges = append(bd.labelChanges, common.Object{
		"op":  "remove_values",
		"key": "plan", // TODO: What key to use?
		"values": common.Array{
			planServicePair.ServiceName + ":" + planServicePair.PlanName,
		},
	})
}

func (bd *BrokerData) addPlan(planServicePair PlanServicePair) {
	bd.plans = append(bd.plans, planServicePair)
	bd.addLabelChangeForNewPlan(planServicePair)
}

func (bd *BrokerData) updateBrokerServerCatalog() {
	if bd.brokerServer != nil {
		bd.brokerServer.Catalog = bd.catalog
	}
}

func (bd *BrokerData) updatePlanAndServiceIDs(ctx *common.TestContext) {
	byBrokerID := query.ByField(query.EqualsOperator, "broker_id", bd.brokerID)
	servicesArray, err := ctx.SMRepository.List(context.TODO(), types.ServiceOfferingType, byBrokerID)
	Expect(err).ToNot(HaveOccurred())

	servicesArrayLen := servicesArray.Len()
	if servicesArrayLen == 0 {
		return
	}
	serviceIDToNameMap := make(map[string]string)
	serviceIDs := make([]string, 0, servicesArrayLen)
	bd.serviceNameToID = make(map[string]string)

	for i := 0; i < servicesArrayLen; i++ {
		serviceObj := servicesArray.ItemAt(i).(*types.ServiceOffering)
		serviceID := serviceObj.ID
		serviceCatalogName := serviceObj.CatalogName
		bd.serviceNameToID[serviceCatalogName] = serviceID
		serviceIDToNameMap[serviceID] = serviceCatalogName
		serviceIDs = append(serviceIDs, serviceID)
	}

	byServiceOfferingID := query.ByField(query.InOperator, "service_offering_id", serviceIDs...)
	plansArray, err := ctx.SMRepository.List(context.TODO(), types.ServicePlanType, byServiceOfferingID)
	Expect(err).ToNot(HaveOccurred())

	bd.planServiceToPlanID = make(map[PlanServicePair]string)
	for i := 0; i < plansArray.Len(); i++ {
		plan := plansArray.ItemAt(i).(*types.ServicePlan)

		serviceName := serviceIDToNameMap[plan.ServiceOfferingID]
		Expect(serviceName).ToNot(BeEmpty())

		bd.planServiceToPlanID[PlanServicePair{
			PlanName:    plan.CatalogName,
			ServiceName: serviceName,
		}] = plan.ID
	}
}

func (bd *BrokerData) ClearLabelChanges() {
	bd.labelChanges = nil
}

func (bd *BrokerData) updateLabels() {
	if bd.isTenantScoped {
		return
	}
	plans := make(common.Array, 0, len(bd.plans))
	for _, planServicePair := range bd.plans {
		plans = append(plans,
			planServicePair.ServiceName+":"+planServicePair.PlanName)
	}
	labels := common.Object{}
	if len(plans) != 0 {
		labels["plans"] = plans
	}
	bd.labels = labels
	bd.labelChanges = nil
}
