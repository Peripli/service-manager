/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package security

import (
	"bytes"
	"fmt"

	"github.com/Peripli/service-manager/pkg/types"
)

// IntegrityProcessor provides functionality to validate and calculate the integrity of a secured object
//go:generate counterfeiter . IntegrityProcessor
type IntegrityProcessor interface {
	ValidateIntegrity(secured types.Secured) bool
	CalculateIntegrity(secured types.Secured) ([32]byte, error)
}

// HashingIntegrityProcessor is an integrity processor that uses a hashing func to calculate the integrity
type HashingIntegrityProcessor struct {
	HashingFunc func(data []byte) [32]byte
}

// CalculateIntegrity calculates the integrity of a secured object using a hashing func
func (h *HashingIntegrityProcessor) CalculateIntegrity(secured types.Secured) ([32]byte, error) {
	var empty [32]byte
	if secured == nil {
		return empty, fmt.Errorf("cannot calculate integrity of nil object")
	}
	integralData := secured.IntegralData()
	if len(integralData) == 0 {
		return empty, nil
	}
	return h.HashingFunc(integralData), nil
}

// ValidateIntegrity validates the integrity of a secured object using a hashing func
func (h *HashingIntegrityProcessor) ValidateIntegrity(secured types.Secured) bool {
	if secured == nil {
		return true
	}
	integralData := secured.IntegralData()
	if len(integralData) == 0 {
		return true
	}
	hashedData := h.HashingFunc(integralData)
	integrity := secured.GetIntegrity()
	return bytes.Equal(hashedData[:], integrity[:])
}
