package healthcheck

import (
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/types"
)

func CheckPlatformsState(platforms []*types.Platform, fatal func(*types.Platform) bool) ( map[string]*health.Health, int,int){
	details := make(map[string]*health.Health)
	inactivePlatforms := 0
	fatalInactivePlatforms:=0
	for _, platform := range platforms {
		if platform.Active {
			details[platform.Name] = health.New().WithStatus(health.StatusUp).
				WithDetail("type", platform.Type)
		} else {
			details[platform.Name] = health.New().WithStatus(health.StatusDown).
				WithDetail("since", platform.LastActive).
				WithDetail("type", platform.Type)
			inactivePlatforms++
			if fatal!=nil &&fatal(platform) {
				fatalInactivePlatforms++
			}

		}
	}
	return details, inactivePlatforms, fatalInactivePlatforms

}