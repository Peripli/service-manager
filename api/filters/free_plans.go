package filters

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/api/broker"
	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

// FreeServicePlansFilter reconciles the state of the free plans offered by all service brokers registered in SM. The
// filter makes sure that a public visibility exists for each free plan present in SM DB.
type FreeServicePlansFilter struct {
	Repository storage.Repository
}

func (fsp *FreeServicePlansFilter) Name() string {
	return "FreePlansFilter"
}

func (fsp *FreeServicePlansFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	response, err := next.Handle(req)
	if err != nil {
		return nil, err
	}
	ctx := req.Context()
	brokerID := req.PathParams[broker.ReqBrokerID]
	log.C(ctx).Debugf("Reconciling free plans for broker with id: %s", brokerID)
	if err := fsp.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		soRepository := fsp.Repository.ServiceOffering()
		vRepository := fsp.Repository.Visibility()

		catalog, err := soRepository.ListWithServicePlansByBrokerID(ctx, brokerID)
		if err != nil {
			return err
		}
		for _, serviceOffering := range catalog {
			for _, servicePlan := range serviceOffering.Plans {
				planID := servicePlan.ID
				isFree := servicePlan.Free
				hasPublicVisibility := false
				visibilitiesForPlan, err := vRepository.ListByServicePlanID(ctx, planID)
				if err != nil {
					return err
				}
				for _, visibility := range visibilitiesForPlan {
					if isFree {
						if visibility.PlatformID == "" {
							hasPublicVisibility = true
							continue
						} else {
							err := vRepository.Delete(ctx, visibility.ID)
							if err != nil {
								return err
							}
						}
					} else {
						if visibility.PlatformID == "" {
							err := vRepository.Delete(ctx, visibility.ID)
							if err != nil {
								return err
							}
						} else {
							continue
						}
					}
				}

				if isFree && !hasPublicVisibility {
					UUID, err := uuid.NewV4()
					if err != nil {
						return fmt.Errorf("could not generate GUID for visibility: %s", err)
					}

					currentTime := time.Now().UTC()
					planID, err := vRepository.Create(ctx, &types.Visibility{
						ID:            UUID.String(),
						ServicePlanID: servicePlan.ID,
						CreatedAt:     currentTime,
						UpdatedAt:     currentTime,
					})
					if err != nil {
						return err
					}

					log.C(ctx).Debugf("Created new public visibility for broker with id %s and plan with id %s", brokerID, planID)
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	log.C(ctx).Debugf("Successfully finished reconciling free plans for broker with id %s", brokerID)
	return response, nil
}

func (fsp *FreeServicePlansFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.BrokersURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPut),
			},
		},
	}
}

// register a broker with one free plan and one paid plan, verify the free plan and the paid plan are created
// test1: add new free plan, update broker, verify plan is created, verify public visibility for the plan is created
// test2: add new paid plan, update broker, verify no public visibility is created for the plan
// test3: verify public visibility exists for the existing free plan, make existing free plan paid, update broker, verify public visibility is no longer present
// test4: verify public visibility does not exist for the existing paid plan, make existing paid plan free, update broker, verify public visibility is created
