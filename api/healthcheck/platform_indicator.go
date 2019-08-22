package healthcheck

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

//TODO: Add Tests

// NewPlatformIndicator returns new health indicator for platforms of given type
func NewPlatformIndicator(ctx context.Context, repository storage.Repository, platformType string) health.Indicator {
	return &platformIndicator{
		ctx:          ctx,
		repository:   repository,
		platformType: platformType,
	}
}

type platformIndicator struct {
	repository   storage.Repository
	ctx          context.Context
	platformType string
}

// Name returns the name of the indicator
func (pi *platformIndicator) Name() string {
	return pi.platformType + "_platforms" // e.g. cf_platforms, k8s_platforms ...
}

// Status returns status of the health check
func (pi *platformIndicator) Status() (interface{}, error) {
	cfCriteria := query.Criterion{
		LeftOp:   "type",
		Operator: query.EqualsOperator,
		RightOp:  []string{pi.platformType},
		Type:     query.FieldQuery,
	}
	objList, err := pi.repository.List(pi.ctx, types.PlatformType, cfCriteria)
	if err != nil {
		return nil, fmt.Errorf("could not fetch platforms health from storage: %v", err)
	}

	details := make(map[string]interface{})
	for i := 0; i < objList.Len(); i++ {
		platform := objList.ItemAt(i).(*types.Platform)
		if platform.Active {
			details[platform.Name] = health.New().WithStatus(health.StatusUp)
		} else {
			details[platform.Name] = health.New().WithStatus(health.StatusDown).WithDetail("since", platform.LastActive)
			err = fmt.Errorf("there is inactive %s platforms", pi.platformType)
		}
	}
	return details, err
}
