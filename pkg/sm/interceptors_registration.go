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

package sm

import (
	"github.com/Peripli/service-manager/storage"
)

type interceptorRegistrationBuilder struct {
	ocip             storage.InterceptorOrder
	registrationFunc func(storage.InterceptorOrder) *ServiceManagerBuilder
}

func (creator *interceptorRegistrationBuilder) AroundTxBefore(name string) *interceptorRegistrationBuilder {
	if creator.ocip.AroundTxPosition.PositionType != storage.PositionNone {
		panic("AroundTxPosition has already been specified")
	}
	creator.ocip.AroundTxPosition = storage.InterceptorPosition{
		Name:         name,
		PositionType: storage.PositionBefore,
	}
	return creator
}

func (creator *interceptorRegistrationBuilder) AroundTxAfter(name string) *interceptorRegistrationBuilder {
	if creator.ocip.AroundTxPosition.PositionType != storage.PositionNone {
		panic("AroundTxPosition has already been specified")
	}
	creator.ocip.AroundTxPosition = storage.InterceptorPosition{
		Name:         name,
		PositionType: storage.PositionAfter,
	}
	return creator
}

func (creator *interceptorRegistrationBuilder) TxBefore(name string) *interceptorRegistrationBuilder {
	if creator.ocip.OnTxPosition.PositionType != storage.PositionNone {
		panic("OnTxPosition has already been specified")
	}
	creator.ocip.OnTxPosition = storage.InterceptorPosition{
		Name:         name,
		PositionType: storage.PositionBefore,
	}
	return creator
}

func (creator *interceptorRegistrationBuilder) TxAfter(name string) *interceptorRegistrationBuilder {
	if creator.ocip.OnTxPosition.PositionType != storage.PositionNone {
		panic("OnTxPosition has already been specified")
	}
	creator.ocip.OnTxPosition = storage.InterceptorPosition{
		Name:         name,
		PositionType: storage.PositionAfter,
	}
	return creator
}

func (creator *interceptorRegistrationBuilder) Before(name string) *interceptorRegistrationBuilder {
	return creator.TxBefore(name).AroundTxBefore(name)
}

func (creator *interceptorRegistrationBuilder) After(name string) *interceptorRegistrationBuilder {
	return creator.TxAfter(name).AroundTxAfter(name)
}

func (creator *interceptorRegistrationBuilder) Register() *ServiceManagerBuilder {
	return creator.registrationFunc(creator.ocip)
}
