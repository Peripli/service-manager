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
	"errors"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"
)

const (
	generatePlatformCredentialsInterceptorName = "CreatePlatformCredentialsInterceptor"
)

type GeneratePlatformCredentialsInterceptorProvider struct {
}

func (c *GeneratePlatformCredentialsInterceptorProvider) Provide() storage.CreateAroundTxInterceptor {
	return &generatePlatformCredentialsInterceptor{}
}

func (c *GeneratePlatformCredentialsInterceptorProvider) Name() string {
	return generatePlatformCredentialsInterceptorName
}

type generatePlatformCredentialsInterceptor struct{}

// AroundTxCreate generates new credentials for the secured object
func (c *generatePlatformCredentialsInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		platform, ok := obj.(*types.Platform)
		if !ok {
			return nil, errors.New("created object is not a platform")
		}
		credentials, err := types.GenerateCredentials()
		if err != nil {
			log.C(ctx).Error("Could not generate credentials for platform")
			return nil, err
		}
		platform.Credentials = credentials

		return h(ctx, obj)
	}
}
