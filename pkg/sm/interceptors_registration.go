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
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

type interceptorRegistrationBuilder struct {
	order            storage.InterceptorOrder
	registrationFunc func(storage.InterceptorOrder) *ServiceManagerBuilder
}

func (irb *interceptorRegistrationBuilder) AroundTxBefore(name string) *interceptorRegistrationBuilder {
	if irb.order.AroundTxPosition.PositionType != storage.PositionNone {
		panic("AroundTxPosition has already been specified")
	}

	irb.order.AroundTxPosition = storage.InterceptorPosition{
		Name:         name,
		PositionType: storage.PositionBefore,
	}

	return irb
}

func (irb *interceptorRegistrationBuilder) AroundTxAfter(name string) *interceptorRegistrationBuilder {
	if irb.order.AroundTxPosition.PositionType != storage.PositionNone {
		panic("AroundTxPosition has already been specified")
	}

	irb.order.AroundTxPosition = storage.InterceptorPosition{
		Name:         name,
		PositionType: storage.PositionAfter,
	}

	return irb
}

func (irb *interceptorRegistrationBuilder) OnTxBefore(name string) *interceptorRegistrationBuilder {
	if irb.order.OnTxPosition.PositionType != storage.PositionNone {
		panic("OnTxPosition has already been specified")
	}

	irb.order.OnTxPosition = storage.InterceptorPosition{
		Name:         name,
		PositionType: storage.PositionBefore,
	}

	return irb
}

func (irb *interceptorRegistrationBuilder) OnTxAfter(name string) *interceptorRegistrationBuilder {
	if irb.order.OnTxPosition.PositionType != storage.PositionNone {
		panic("OnTxPosition has already been specified")
	}

	irb.order.OnTxPosition = storage.InterceptorPosition{
		Name:         name,
		PositionType: storage.PositionAfter,
	}

	return irb
}

func (irb *interceptorRegistrationBuilder) Before(name string) *interceptorRegistrationBuilder {
	return irb.OnTxBefore(name).AroundTxBefore(name)
}

func (irb *interceptorRegistrationBuilder) After(name string) *interceptorRegistrationBuilder {
	return irb.OnTxAfter(name).AroundTxAfter(name)
}

func (irb *interceptorRegistrationBuilder) Register() *ServiceManagerBuilder {
	return irb.registrationFunc(irb.order)
}
