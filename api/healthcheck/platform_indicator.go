/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package healthcheck

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

// NewPlatformIndicator returns new health indicator for platforms of given type
func NewPlatformIndicator(ctx context.Context, repository storage.Repository, fatal func(*types.Platform) bool) health.Indicator {
	if fatal == nil {
		fatal = func(platform *types.Platform) bool {
			return true
		}
	}
	return &platformIndicator{
		ctx:        ctx,
		repository: repository,
		fatal:      fatal,
	}
}

type platformIndicator struct {
	repository storage.Repository
	ctx        context.Context
	fatal      func(*types.Platform) bool
}

// Name returns the name of the indicator
func (pi *platformIndicator) Name() string {
	return health.PlatformsIndicatorName
}

// Status returns status of the health check
func (pi *platformIndicator) Status() (interface{}, error) {

	criteria := []query.Criterion{
		query.ByField(query.NotEqualsOperator, "id", types.SMPlatform),
		query.ByField(query.EqualsOperator, "technical", "false"),
	}

	objList, err := pi.repository.List(pi.ctx, types.PlatformType, criteria...)
	if err != nil {
		return nil, fmt.Errorf("could not fetch platforms health from storage: %v", err)
	}
	platforms := objList.(*types.Platforms).Platforms

	details := make(map[string]*health.Health)
	inactivePlatforms := 0
	fatalInactivePlatforms := 0
	for _, platform := range platforms {
		if platform.Active {
			details[platform.Name] = health.New().WithStatus(health.StatusUp).
				WithDetail("type", platform.Type)
		} else {
			details[platform.Name] = health.New().WithStatus(health.StatusDown).
				WithDetail("since", platform.LastActive).
				WithDetail("type", platform.Type)
			inactivePlatforms++
			if pi.fatal(platform) {
				fatalInactivePlatforms++
			}
		}
	}

	if fatalInactivePlatforms > 0 {
		err = fmt.Errorf("there are %d inactive platforms %d of them are fatal", inactivePlatforms, fatalInactivePlatforms)
	}

	return details, err
}
