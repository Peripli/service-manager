package filters

import (
	"context"
	"errors"
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/web"
)

const BrokersVisibilityFilterName = "BrokersFilterByVisibility"

func NewBrokersFilterByVisibility(repository storage.Repository) *BrokersFilterByVisibility {
	return &BrokersFilterByVisibility{
		repository: repository,
	}
}

type BrokersFilterByVisibility struct {
	repository storage.Repository
}

func (vf *BrokersFilterByVisibility) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	userCtx, ok := web.UserFromContext(ctx)
	if !ok {
		return nil, errors.New("No user found")
	}
	if userCtx.AuthenticationType != web.Basic {
		log.C(ctx).Debugf("Authentication is %s, not basic so proceed without visibility filter criteria", userCtx.AuthenticationType)
		return next.Handle(req)
	}
	platform := &types.Platform{}
	if err := userCtx.Data(platform); err != nil {
		return nil, err
	}

	zeroResult := false

	planQuery, err := plansCriteria(ctx, vf.repository, platform.ID)
	if err != nil {
		return nil, err
	}
	if planQuery == nil {
		zeroResult = true
	}

	var servicesQuery *query.Criterion
	if !zeroResult {
		servicesQuery, err = servicesCriteria(ctx, vf.repository, planQuery)
		if err != nil {
			return nil, err
		}
		if servicesQuery == nil {
			zeroResult = true
		}
	}

	if !zeroResult {
		objectID := req.PathParams["id"]
		brokersQuery, err := brokersCriteria(ctx, vf.repository, servicesQuery)
		if err != nil {
			return nil, err
		}

		hasID := false
		if brokersQuery != nil {
			for _, brokerID := range brokersQuery.RightOp {
				if brokerID == objectID {
					hasID = true
					break
				}
			}
		}

		if brokersQuery == nil || (!hasID && objectID != "") {
			zeroResult = true
		} else if objectID == "" {
			ctx = query.ContextWithCriteria(ctx, []query.Criterion{*brokersQuery})
		}
	}

	if zeroResult {
		// return util.NewJSONResponse(http.StatusOK, types.ServiceBrokers{
		// 	ServiceBrokers: nil,
		// })
		ctx = query.ContextWithCriteria(ctx, []query.Criterion{query.LimitResultBy(0)})
	}

	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}

func brokersCriteria(ctx context.Context, repository storage.Repository, serviceQuery *query.Criterion) (*query.Criterion, error) {
	objectList, err := repository.List(ctx, types.ServiceOfferingType, *serviceQuery)
	if err != nil {
		return nil, err
	}
	offerings := objectList.(*types.ServiceOfferings)
	if offerings.Len() < 1 {
		return nil, nil
	}
	brokerIDs := make([]string, 0, offerings.Len())
	for _, p := range offerings.ServiceOfferings {
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
	objectList, err := repository.List(ctx, types.VisibilityType, query.ByField(query.EqualsOrNilOperator, "platform_id", platformID))
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

func (vf *BrokersFilterByVisibility) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBrokersURL + "/**"),
				web.Methods(http.MethodGet, http.MethodPatch, http.MethodPut, http.MethodDelete),
			},
		},
	}
}

func (vf *BrokersFilterByVisibility) Name() string {
	return BrokersVisibilityFilterName
}
