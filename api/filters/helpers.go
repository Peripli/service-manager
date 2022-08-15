package filters

import (
	"context"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

func brokersCriteria(ctx context.Context, repository storage.Repository, servicesQuery *query.Criterion) (*query.Criterion, error) {
	objectList, err := repository.List(ctx, types.ServiceOfferingType, *servicesQuery)
	if err != nil {
		return nil, err
	}
	services := objectList.(*types.ServiceOfferings)
	if services.Len() < 1 {
		return nil, nil
	}
	brokerIDs := make([]string, 0, services.Len())
	for _, p := range services.ServiceOfferings {
		brokerIDs = append(brokerIDs, p.BrokerID)
	}
	c := query.ByField(query.InOperator, "id", brokerIDs...)
	return &c, nil
}

func servicesCriteria(ctx context.Context, repository storage.Repository, planQuery *query.Criterion) (*query.Criterion, error) {
	objectList, err := repository.List(ctx, types.ServicePlanType, *planQuery)
	if err != nil {
		return nil, err
	}
	plans := objectList.(*types.ServicePlans)
	if plans.Len() < 1 {
		return nil, nil
	}
	serviceIDs := make([]string, 0, plans.Len())
	for _, p := range plans.ServicePlans {
		serviceIDs = append(serviceIDs, p.ServiceOfferingID)
	}
	c := query.ByField(query.InOperator, "id", serviceIDs...)
	return &c, nil
}

func plansCriteria(ctx context.Context, repository storage.Repository, platformID string) (*query.Criterion, error) {
	objectList, err := repository.ListNoLabels(ctx, types.VisibilityType, query.ByField(query.EqualsOrNilOperator, "platform_id", platformID))
	if err != nil {
		return nil, err
	}
	visibilityList := objectList.(*types.Visibilities)
	if visibilityList.Len() < 1 {
		return nil, nil
	}
	planIDs := make([]string, 0, visibilityList.Len())
	for _, vis := range visibilityList.Visibilities {
		planIDs = append(planIDs, vis.ServicePlanID)
	}
	c := query.ByField(query.InOperator, "id", planIDs...)
	return &c, nil
}

func getSharedProperty(reqBody []byte) *bool {
	var reqServiceInstance types.ServiceInstance
	err := util.BytesToObjectNoLabels(reqBody, &reqServiceInstance)
	if err != nil {
		return nil
	}
	return reqServiceInstance.Shared
}
