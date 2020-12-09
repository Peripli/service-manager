package healthcheck

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func NewMonioredPlatformsIndicator(ctx context.Context, repository storage.Repository, monitoredPlatformsThreshold int) health.Indicator {
	return &monitoredPlatformsIndicator{
		ctx:                         ctx,
		repository:                  repository,
		monitoredPlatformsThreshold: monitoredPlatformsThreshold,
	}
}

type monitoredPlatformsIndicator struct {
	repository                  storage.Repository
	ctx                         context.Context
	monitoredPlatformsThreshold int
}

const MonitoredPlatformsHealthIndicatorName string = "monitored_platforms"

// Name returns the name of the indicator
func (pi *monitoredPlatformsIndicator) Name() string {
	return MonitoredPlatformsHealthIndicatorName
}

// Status returns status of the health check
func (pi *monitoredPlatformsIndicator) Status() (interface{}, error) {
	monitoredPlatforms, err := pi.repository.QueryForList(pi.ctx, types.PlatformType, storage.QueryByExistingLabel, map[string]interface{}{"key": types.Monitored})
	if err != nil {
		return nil, fmt.Errorf("could not fetch monitored platforms health from storage: %v", err)
	}
	platforms := monitoredPlatforms.(*types.Platforms).Platforms
	details, inactivePlatforms, _ := CheckPlatformsState(platforms,nil)
	if len(platforms) > 0 {
		currentThreshold := (inactivePlatforms / len(platforms)) * 100
		if currentThreshold >= pi.monitoredPlatformsThreshold {
			err = fmt.Errorf("%d % of the platforms are failing", currentThreshold)
		}
	}
	return details, err
}
