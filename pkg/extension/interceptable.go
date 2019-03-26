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

type Interceptable interface {
	InterceptsType() types.ObjectType

	AddCreateInterceptorProviders(providers ...CreateInterceptorProvider)
	AddUpdateInterceptorProviders(providers ...UpdateInterceptorProvider)
	AddDeleteInterceptorProviders(providers ...DeleteInterceptorProvider)
}

type PositionType string

const (
	PositionBefore PositionType = "before"
	PositionAfter  PositionType = "after"
)

type OrderedProviderImpl struct {
	PositionTypeTx  PositionType
	PositionTypeAPI PositionType
	NameTx          string
	NameAPI         string
}

func (opi *OrderedProviderImpl) PositionTransaction() (PositionType, string) {
	return opi.PositionTypeTx, opi.NameTx
}

func (opi *OrderedProviderImpl) PositionAPI() (PositionType, string) {
	return opi.PositionTypeAPI, opi.NameAPI
}

type Ordered interface {
	PositionTransaction() (PositionType, string)
	PositionAPI() (PositionType, string)
}

type Named interface {
	Name() string
}
