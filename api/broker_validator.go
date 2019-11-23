package api

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/Peripli/service-manager/storage"
)

// BrokerValidator is a type of ResourceValidator
type BrokerValidator struct {
	DefaultResourceValidator
	CatalogFetcher func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error)
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
		return err
	}

	broker := object.(*types.ServiceBroker)

	currentCatalog := struct {
		Services []*types.ServiceOffering `json:"services"`
	}{}
	if err := util.BytesToObject(broker.Catalog, &currentCatalog); err != nil {
		return err
	}

	catalogBytes, err := bv.CatalogFetcher(ctx, broker)
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
		log.C(ctx).Debugf("Found service instances associated with plans removed from catalog of broker with ID (%s): %s", object.GetID(), instanceIDs)
		return &util.ErrExistingReferenceEntityStorage{
			Entity:          string(types.ServicePlanType),
			ViolationEntity: string(types.ServiceInstanceType),
			ViolationIDs:    instanceIDs,
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
		log.C(ctx).Debugf("Found service instances associated with broker with ID (%s): %s", object.GetID(), instanceIDs)
		return &util.ErrExistingReferenceEntityStorage{
			Entity:          string(types.ServicePlanType),
			ViolationEntity: string(types.ServiceInstanceType),
			ViolationIDs:    instanceIDs,
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
	instanceIDs := make([]string, 0)
	for _, catalogPlanID := range catalogPlanIDs {
		byCatalogPlanID := query.ByField(query.EqualsOperator, "catalog_id", catalogPlanID)
		objectList, err := repository.List(ctx, types.ServicePlanType, byCatalogPlanID)
		if err != nil {
			return nil, util.HandleStorageError(err, string(types.ServicePlanType))
		}

		for i := 0; i < objectList.Len(); i++ {
			byPlanID := query.ByField(query.EqualsOperator, "service_plan_id", objectList.ItemAt(i).GetID())
			ids, err := retrieveServiceInstanceIDsByCriteria(ctx, repository, byPlanID)
			if err != nil {
				return nil, util.HandleStorageError(err, string(types.ServiceInstanceType))
			}

			instanceIDs = append(instanceIDs, ids...)
		}
	}
	return instanceIDs, nil
}

func retrieveServiceInstanceIDsByBrokerID(ctx context.Context, repository storage.Repository, brokerID string) ([]string, error) {
	instanceIDs := make([]string, 0)

	byBrokerID := query.ByField(query.EqualsOperator, "broker_id", brokerID)
	offeringList, err := repository.List(ctx, types.ServiceOfferingType, byBrokerID)
	if err != nil {
		return nil, util.HandleStorageError(err, string(types.ServiceOfferingType))
	}

	for i := 0; i < offeringList.Len(); i++ {
		byOfferingID := query.ByField(query.EqualsOperator, "service_offering_id", offeringList.ItemAt(i).GetID())
		planList, err := repository.List(ctx, types.ServicePlanType, byOfferingID)
		if err != nil {
			return nil, util.HandleStorageError(err, string(types.ServicePlanType))
		}

		for j := 0; j < planList.Len(); j++ {
			byPlanID := query.ByField(query.EqualsOperator, "service_plan_id", planList.ItemAt(j).GetID())
			ids, err := retrieveServiceInstanceIDsByCriteria(ctx, repository, byPlanID)
			if err != nil {
				return nil, util.HandleStorageError(err, string(types.ServiceInstanceType))
			}

			instanceIDs = append(instanceIDs, ids...)
		}
	}

	return instanceIDs, nil
}
