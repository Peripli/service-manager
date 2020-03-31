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
