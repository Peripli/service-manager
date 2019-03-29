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

package extension

import "github.com/Peripli/service-manager/pkg/types"

// Interceptable is an interface which Controllers implement to support interceptors
type Interceptable interface {
	InterceptsType() types.ObjectType

	AddCreateInterceptorProviders(providers ...CreateInterceptorProvider)
	AddUpdateInterceptorProviders(providers ...UpdateInterceptorProvider)
	AddDeleteInterceptorProviders(providers ...DeleteInterceptorProvider)
}

// PositionType could be "before", "after" or "none"
type PositionType string

const (
	// PositionNone states that a position is not set and the item will be appended
	PositionNone PositionType = "none"

	// PositionBefore states that a position should be calculated before another position
	PositionBefore PositionType = "before"

	// PositionAfter states that a position should be calculated after another position
	PositionAfter PositionType = "after"
)

// OrderedProviderImpl is an implementation of Ordered interface used for ordering interceptors
type OrderedProviderImpl struct {
	PositionTypeTx  PositionType
	PositionTypeAPI PositionType
	NameTx          string
	NameAPI         string
}

// PositionTransaction returns the position of the interceptor transaction hook
func (opi *OrderedProviderImpl) PositionTransaction() (PositionType, string) {
	if opi.NameTx == "" {
		return PositionNone, ""
	}
	return opi.PositionTypeTx, opi.NameTx
}

// PositionAPI returns the position of the interceptor out of transaction hook
func (opi *OrderedProviderImpl) PositionAPI() (PositionType, string) {
	if opi.NameAPI == "" {
		return PositionNone, ""
	}
	return opi.PositionTypeAPI, opi.NameAPI
}

// Ordered interface for positioning interceptors
type Ordered interface {
	PositionTransaction() (PositionType, string)
	PositionAPI() (PositionType, string)
}

// Named interface for named entities
//go:generate counterfeiter . Named
type Named interface {
	Name() string
}
