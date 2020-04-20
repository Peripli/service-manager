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
	"github.com/tidwall/gjson"
)

// SupportsPlatform determines whether a specific platform type is among the ones that a plan supports
func (e *ServicePlan) SupportsPlatformType(platformType string) bool {
	if platformType == SMPlatform {
		platformType = GetSMSupportedPlatformType()
	}

	platformTypes := e.SupportedPlatformTypes()

	return len(platformTypes) == 0 || slice.StringsAnyEquals(platformTypes, platformType)
}

// SupportsPlatformInstance determines whether a specific platform instance is among the ones that a plan supports
func (e *ServicePlan) SupportsPlatformInstance(platform Platform) bool {
	platformNames := e.SupportedPlatformNames()

	if len(platformNames) == 0 {
		return e.SupportsPlatformType(platform.Type)
	} else {
		return slice.StringsAnyEquals(platformNames, platform.Name)
	}
}

// SupportsAllPlatforms determines whether the plan supports all platforms
func (e *ServicePlan) SupportsAllPlatforms() bool {
	return len(e.SupportedPlatformNames()) == 0 && len(e.SupportedPlatformTypes()) == 0
}

// SupportedPlatformTypes returns the supportedPlatforms provided in a plan's metadata (if a value is provided at all).
// If there are no supported platforms, nil is returned denoting that the plan is available to platforms of all types.
func (e *ServicePlan) SupportedPlatformTypes() []string {
	return e.metadataPropertyAsStringArray("supportedPlatforms")
}

// SupportedPlatformNames returns the supportedPlatformNames provided in a plan's metadata (if a value is provided at all).
// If there are no supported platforms names, nil is returned
func (e *ServicePlan) SupportedPlatformNames() []string {
	return e.metadataPropertyAsStringArray("supportedPlatformNames")
}

func (e *ServicePlan) metadataPropertyAsStringArray(propertyKey string) []string {
	propertyValue := gjson.GetBytes(e.Metadata, propertyKey)
	if !propertyValue.IsArray() || len(propertyValue.Array()) == 0 {
		return []string{}
	}
	array := propertyValue.Array()
	result := make([]string, len(array))

	for i, p := range propertyValue.Array() {
		result[i] = p.String()
	}
	return result
}
