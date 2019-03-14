/*
 * Copyright 2018 The Service Manager Authors
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
	"errors"
	"fmt"

	"github.com/Peripli/service-manager/pkg/util"
)

//go:generate smgen api Visibility
// Visibility struct
type Visibility struct {
	Base
	PlatformID string `json:"platform_id"`
	Labels     Labels `json:"labels,omitempty"`

	ServicePlanID string `json:"service_plan_id"`
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (v *Visibility) Validate() error {
	if v.ServicePlanID == "" {
		return errors.New("missing visibility service plan id")
	}
	if util.HasRFC3986ReservedSymbols(v.ID) {
		return fmt.Errorf("%s contains invalid character(s)", v.ID)
	}
	if err := v.Labels.Validate(); err != nil {
		return err
	}
	return nil
}
