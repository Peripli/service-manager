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

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
)

//go:generate smgen api Visibility
// Visibility struct
type Visibility struct {
	Base
	PlatformID    string `json:"platform_id"`
	ServicePlanID string `json:"service_plan_id"`
}

func (e *Visibility) Equals(obj Object) bool {
	if !Equals(e, obj) {
		return false
	}

	visibility := obj.(*Visibility)
	if e.PlatformID != visibility.PlatformID ||
		e.ServicePlanID != visibility.ServicePlanID {
		return false
	}

	return true
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (e *Visibility) Validate() error {
	if e.ServicePlanID == "" {
		return errors.New("missing visibility service plan id")
	}
	if util.HasRFC3986ReservedSymbols(e.ID) {
		return fmt.Errorf("%s contains invalid character(s)", e.ID)
	}
	if err := e.Labels.Validate(); err != nil {
		return err
	}
	return nil
}
