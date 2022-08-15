package healthcheck

import (
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/health"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

func CheckPlatformsState(platforms []*types.Platform, fatal func(*types.Platform) bool) (map[string]*health.Health, int, int) {
	details := make(map[string]*health.Health)
	inactivePlatforms := 0
	fatalInactivePlatforms := 0
	for _, platform := range platforms {
		if platform.ID == types.SMPlatform ||
			(!platform.Active && platform.LastActive.IsZero()) {
			continue
		}
		healthObj := health.New().WithDetail("type", platform.Type)
		if platform.Active {
			healthObj = healthObj.WithStatus(health.StatusUp)
		} else {
			inactivePlatforms++
			healthObj.WithStatus(health.StatusDown).WithDetail("since", platform.LastActive)
			if fatal != nil && fatal(platform) {
				fatalInactivePlatforms++
			}
		}

		details[platform.Name] = healthObj

	}
	return details, inactivePlatforms, fatalInactivePlatforms

}
