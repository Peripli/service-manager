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
)

// IntegralObject interface indicates that an objects needs to be processed with regards to its integrity
type IntegralObject interface {
	IntegralData() []byte
	SetIntegrity([]byte)
	GetIntegrity() []byte
}

// IntegrityProcessor provides functionality to validate and calculate the integrity of an integral object
//go:generate counterfeiter . IntegrityProcessor
type IntegrityProcessor interface {
	ValidateIntegrity(integral IntegralObject) bool
	CalculateIntegrity(integral IntegralObject) ([]byte, error)
}

// HashingIntegrityProcessor is an integrity processor that uses a hashing func to calculate the integrity
type HashingIntegrityProcessor struct {
	HashingFunc func(data []byte) []byte
}

// CalculateIntegrity calculates the integrity of an integral object using a hashing func
func (h *HashingIntegrityProcessor) CalculateIntegrity(integral IntegralObject) ([]byte, error) {
	if integral == nil {
		return nil, fmt.Errorf("cannot calculate integrity of nil object")
	}
	integralData := integral.IntegralData()
	if len(integralData) == 0 {
		return []byte{}, nil
	}
	return h.HashingFunc(integralData), nil
}

// ValidateIntegrity validates the integrity of an integral object using a hashing func
func (h *HashingIntegrityProcessor) ValidateIntegrity(integral IntegralObject) bool {
	if integral == nil {
		return true
	}
	integralData := integral.IntegralData()
	if len(integralData) == 0 {
		return true
	}
	hashedData := h.HashingFunc(integralData)
	integrity := integral.GetIntegrity()
	return bytes.Equal(hashedData, integrity)
}
