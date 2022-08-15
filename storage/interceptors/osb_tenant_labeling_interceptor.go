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

package interceptors

import (
	"context"
	"encoding/json"

	"github.com/tidwall/gjson"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

const (
	ServiceInstanceCreateInterceptorName = "ServiceInstanceCreateInterceptor"
	ServiceBindingCreateInterceptorName  = "ServiceBindingCreateInterceptor"
)

type osbTenantLabelingInterceptor struct {
	tenantIdentifier string
	interceptorName  string
	extractContext   func(obj types.Object) json.RawMessage
}

func (otl *osbTenantLabelingInterceptor) Name() string {
	return otl.interceptorName
}

func NewOSBServiceInstanceTenantLabelingInterceptor(tenantIdentifier string) *osbTenantLabelingInterceptor {
	return &osbTenantLabelingInterceptor{
		tenantIdentifier: tenantIdentifier,
		interceptorName:  ServiceInstanceCreateInterceptorName,
		extractContext: func(obj types.Object) json.RawMessage {
			return obj.(*types.ServiceInstance).Context
		},
	}
}

func NewOSBBindingTenantLabelingInterceptor(tenantIdentifier string) *osbTenantLabelingInterceptor {
	return &osbTenantLabelingInterceptor{
		tenantIdentifier: tenantIdentifier,
		interceptorName:  ServiceBindingCreateInterceptorName,
		extractContext: func(obj types.Object) json.RawMessage {
			return obj.(*types.ServiceBinding).Context
		},
	}
}

func (otl *osbTenantLabelingInterceptor) Provide() storage.CreateOnTxInterceptor {
	return otl
}

func (otl *osbTenantLabelingInterceptor) OnTxCreate(h storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, storage storage.Repository, obj types.Object) (types.Object, error) {
		labels := obj.GetLabels()
		if labels == nil {
			labels = types.Labels{}
		}

		if _, ok := labels[otl.tenantIdentifier]; ok {
			log.C(ctx).Debugf("Label %s is already set on %s", otl.tenantIdentifier, obj.GetType())
			return h(ctx, storage, obj)
		}

		objectContext := otl.extractContext(obj)
		tenantID := gjson.GetBytes(objectContext, otl.tenantIdentifier)
		if !tenantID.Exists() {
			log.C(ctx).Debugf("Could not add %s label to %s with id %s. Label not found in OSB context.", otl.tenantIdentifier, obj.GetType(), obj.GetID())
			return h(ctx, storage, obj)
		}
		labels[otl.tenantIdentifier] = []string{tenantID.String()}
		obj.SetLabels(labels)
		log.C(ctx).Debugf("Adding label %s=%s to %s with id %s.", otl.tenantIdentifier, tenantID, obj.GetType(), obj.GetID())

		return h(ctx, storage, obj)
	}
}
