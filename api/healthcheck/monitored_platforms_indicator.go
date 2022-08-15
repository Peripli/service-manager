package healthcheck

import (
	"context"
	"fmt"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/health"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

func NewMonitoredPlatformsIndicator(ctx context.Context, repository storage.Repository, monitoredPlatformsThreshold int) health.Indicator {
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

// Name returns the name of the indicator
func (pi *monitoredPlatformsIndicator) Name() string {
	return health.MonitoredPlatformsHealthIndicatorName
}

// Status returns status of the health check
func (pi *monitoredPlatformsIndicator) Status() (interface{}, error) {
	objList, err := pi.repository.QueryForList(pi.ctx, types.PlatformType, storage.QueryByExistingLabel, map[string]interface{}{"key": types.Monitored})
	if err != nil {
		return nil, fmt.Errorf("unable to query for monitored platforms: %v", err)
	}
	monitoredPlatforms := objList.(*types.Platforms).Platforms
	details, inactivePlatforms, _ := CheckPlatformsState(monitoredPlatforms, nil)
	return details, isHealthy(monitoredPlatforms, inactivePlatforms, pi, err)
}

func isHealthy(monitoredPlatforms []*types.Platform, inactivePlatforms int, pi *monitoredPlatformsIndicator, err error) error {
	if len(monitoredPlatforms) > 0 {
		currentThreshold := (inactivePlatforms * 100.00 / len(monitoredPlatforms))
		if currentThreshold >= pi.monitoredPlatformsThreshold {
			err = fmt.Errorf("%d%% of the monitored platforms are failing", currentThreshold)
		}
	}
	return err
}
