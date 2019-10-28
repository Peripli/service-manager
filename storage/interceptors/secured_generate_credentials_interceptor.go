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

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"
)

const (
	generateCredentialsInterceptorName = "CreateCredentialsInterceptor"
)

type GenerateCredentialsInterceptorProvider struct {
}

func (c *GenerateCredentialsInterceptorProvider) Provide() storage.CreateAroundTxInterceptor {
	return &generateCredentialsInterceptor{}
}

func (c *GenerateCredentialsInterceptorProvider) Name() string {
	return generateCredentialsInterceptorName
}

type generateCredentialsInterceptor struct{}

// AroundTxCreate generates new credentials for the secured object
func (c *generateCredentialsInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		credentials, err := types.GenerateCredentials()
		if err != nil {
			log.C(ctx).Error("Could not generate credentials for platform")
			return nil, err
		}
		(obj.(types.Secured)).SetCredentials(credentials)

		return h(ctx, obj)
	}
}
