package api

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/Peripli/service-manager/storage"
	"net/http"
)

// BrokerValidator is a type of ResourceValidator
type BrokerValidator struct {
	DefaultResourceValidator
	FetchCatalog func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error)
}

// ValidateUpdate ensures that there are no service instances associated with a service plan
// in the cases when a broker's catalog gets updated by removing that service plan
func (bv *BrokerValidator) ValidateUpdate(ctx context.Context, repository storage.Repository, object types.Object) error {
	_, ok := object.(*types.ServiceBroker)
	if !ok {
		log.C(ctx).Debugf("Provided object is of type %s. Cannot validate broker deletion.", object.GetType())
		return ErrIncompatibleObjectType
	}

	if err := object.Validate(); err != nil {
		return &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: err.Error(),
			StatusCode:  http.StatusBadRequest,
		}
	}

	broker := object.(*types.ServiceBroker)

	currentCatalog := struct {
		Services []*types.ServiceOffering `json:"services"`
	}{}
	if err := util.BytesToObject(broker.Catalog, &currentCatalog); err != nil {
		return err
	}

	catalogBytes, err := bv.FetchCatalog(ctx, broker)
	if err != nil {
		return err
	}
	latestCatalog := struct {
		Services []*types.ServiceOffering `json:"services"`
	}{}
	if err := util.BytesToObject(catalogBytes, &latestCatalog); err != nil {
		return err
	}

	currentCatalogPlanIDs := retrieveCatalogPlanIDs(currentCatalog.Services)
	latestCatalogPlanIDs := retrieveCatalogPlanIDs(latestCatalog.Services)

	removedCatalogPlanIDs := make([]string, 0)
	for _, catalogPlanID := range currentCatalogPlanIDs {
		if !slice.StringsAnyEquals(latestCatalogPlanIDs, catalogPlanID) {
			removedCatalogPlanIDs = append(removedCatalogPlanIDs, catalogPlanID)
		}
	}

	if len(removedCatalogPlanIDs) > 0 {
		log.C(ctx).Debugf("Fetching service instances for plans removed from catalog of broker with ID (%s)", object.GetID())
	}
	instanceIDs, err := retrieveServiceInstanceIDsByCatalogPlanIDs(ctx, repository, removedCatalogPlanIDs)
	if err != nil {
		return err
	}

	if len(instanceIDs) > 0 {
		log.C(ctx).Infof("Found service instances associated with plans removed from catalog of broker with ID (%s): %s", object.GetID(), instanceIDs)
		return &util.ErrForeignKeyViolation{
			Entity:             types.ServicePlanType.String(),
			ReferenceEntity:    types.ServiceInstanceType.String(),
			ReferenceEntityIDs: instanceIDs,
		}
	}

	log.C(ctx).Debugf("No service instances associated with plans removed from catalog of broker with ID (%s) were found. Broker update can continue...", object.GetID())
	return nil
}

// ValidateDelete ensures that there are no service instances associated with a service plan of the broker that's about to be deleted
func (bv *BrokerValidator) ValidateDelete(ctx context.Context, repository storage.Repository, object types.Object) error {
	_, ok := object.(*types.ServiceBroker)
	if !ok {
		log.C(ctx).Debugf("Provided object is of type %s. Cannot validate broker deletion.", object.GetType())
		return ErrIncompatibleObjectType
	}

	log.C(ctx).Debugf("Fetching service instances for broker with ID (%s)", object.GetID())
	instanceIDs, err := retrieveServiceInstanceIDsByBrokerID(ctx, repository, object.GetID())
	if err != nil {
		return err
	}

	if len(instanceIDs) > 0 {
		log.C(ctx).Infof("Found service instances associated with broker with ID (%s): %s", object.GetID(), instanceIDs)
		return &util.ErrForeignKeyViolation{
			Entity:             types.ServicePlanType.String(),
			ReferenceEntity:    types.ServiceInstanceType.String(),
			ReferenceEntityIDs: instanceIDs,
		}
	}

	log.C(ctx).Debugf("No service instances associated with broker with ID (%s) were found. Broker deletion can continue...", object.GetID())
	return nil
}

func retrieveCatalogPlanIDs(services []*types.ServiceOffering) []string {
	catalogPlanIDs := make([]string, 0)
	for _, svc := range services {
		for _, plan := range svc.Plans {
			catalogPlanIDs = append(catalogPlanIDs, plan.ID)
		}
	}
	return catalogPlanIDs
}

func retrieveServiceInstanceIDsByCatalogPlanIDs(ctx context.Context, repository storage.Repository, catalogPlanIDs []string) ([]string, error) {
	if len(catalogPlanIDs) == 0 {
		return []string{}, nil
	}

	byCatalogPlanIDs := query.ByField(query.InOperator, "catalog_id", catalogPlanIDs...)
	instanceIDs, err := retrieveServiceInstanceIDsByPlanCriteria(ctx, repository, byCatalogPlanIDs)
	if err != nil {
		return nil, err
	}

	return instanceIDs, nil
}

func retrieveServiceInstanceIDsByBrokerID(ctx context.Context, repository storage.Repository, brokerID string) ([]string, error) {
	byBrokerID := query.ByField(query.EqualsOperator, "broker_id", brokerID)
	offeringList, err := repository.List(ctx, types.ServiceOfferingType, byBrokerID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceOfferingType.String())
	}

	if offeringList.Len() == 0 {
		return []string{}, nil
	}

	offeringIDs := make([]string, 0)
	for i := 0; i < offeringList.Len(); i++ {
		offeringIDs = append(offeringIDs, offeringList.ItemAt(i).GetID())
	}

	byOfferingIDs := query.ByField(query.InOperator, "service_offering_id", offeringIDs...)
	instanceIDs, err := retrieveServiceInstanceIDsByPlanCriteria(ctx, repository, byOfferingIDs)
	if err != nil {
		return nil, err
	}

	return instanceIDs, nil
}

func retrieveServiceInstanceIDsByPlanCriteria(ctx context.Context, repository storage.Repository, planCriteria ...query.Criterion) ([]string, error) {
	planList, err := repository.List(ctx, types.ServicePlanType, planCriteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}

	if planList.Len() == 0 {
		return []string{}, nil
	}

	planIDs := make([]string, 0)
	for i := 0; i < planList.Len(); i++ {
		planIDs = append(planIDs, planList.ItemAt(i).GetID())
	}

	byPlanIDs := query.ByField(query.InOperator, "service_plan_id", planIDs...)
	instanceIDs, err := retrieveServiceInstanceIDsByCriteria(ctx, repository, byPlanIDs)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	return instanceIDs, nil
}
