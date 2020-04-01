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

package types

import (
	"github.com/Peripli/service-manager/pkg/util/slice"
)

// SupportsPlatform determines whether a specific platform type is among the ones that a plan supports
func (e *ServicePlan) SupportsPlatformType(platform string) bool {
	platformTypes := e.SupportedPlatformTypes()

	return platformTypes == nil || slice.StringsAnyEquals(platformTypes, platform)
}

// SupportsPlatformInstance determines whether a specific platform instance is among the ones that a plan supports
func (e *ServicePlan) SupportsPlatformInstance(platform Platform) bool {
	platformNames := e.SupportedPlatformNames()

	if platformNames == nil {
		return e.SupportsPlatformType(platform.Type)
	} else {
		return slice.StringsAnyEquals(platformNames, platform.Name)
	}
}

// SupportsAllPlatforms determines whether the plan supports all platforms
func (e *ServicePlan) SupportsAllPlatforms() bool {
	return len(e.SupportedPlatformNames()) == 0 && len(e.SupportedPlatformTypes()) == 0
}
