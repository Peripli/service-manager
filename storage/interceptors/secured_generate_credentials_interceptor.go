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
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"
)

const (
	generatePlatformCredentialsInterceptorName   = "CreatePlatformCredentialsInterceptor"
	regeneratePlatformCredentialsInterceptorName = "UpdatePlatformCredentialsInterceptor"
)

type GeneratePlatformCredentialsInterceptorProvider struct {
}

type RegeneratePlatformCredentialsInterceptorProvider struct {
}

func (c *GeneratePlatformCredentialsInterceptorProvider) Provide() storage.CreateAroundTxInterceptor {
	return &generatePlatformCredentialsInterceptor{}
}

func (c *GeneratePlatformCredentialsInterceptorProvider) Name() string {
	return generatePlatformCredentialsInterceptorName
}

func (c *RegeneratePlatformCredentialsInterceptorProvider) Provide() storage.UpdateAroundTxInterceptor {
	return &generatePlatformCredentialsInterceptor{}
}

func (c *RegeneratePlatformCredentialsInterceptorProvider) Name() string {
	return regeneratePlatformCredentialsInterceptorName
}

type generatePlatformCredentialsInterceptor struct{}

// AroundTxCreate generates new credentials for the secured object
func (c *generatePlatformCredentialsInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		platform, ok := obj.(*types.Platform)
		if !ok {
			return nil, errors.New("created object is not a platform")
		}

		if platform.Technical {
			return h(ctx, obj)
		}

		if err := generateCredentials(ctx, platform); err != nil {
			return nil, err
		}
		return h(ctx, obj)
	}
}

func (c *generatePlatformCredentialsInterceptor) AroundTxUpdate(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return func(ctx context.Context, obj types.Object, labelChanges ...*types.LabelChange) (types.Object, error) {
		platform, ok := obj.(*types.Platform)
		if !ok {
			return nil, errors.New("created object is not a platform")
		}

		if platform.Technical {
			return h(ctx, platform, labelChanges...)
		}

		if web.IsGeneratePlatformCredentialsRequired(ctx) {
			log.C(ctx).Infof("Generating credentials for platform %s, current credentials are active [%v]", platform.ID, platform.CredentialsActive)
			if platform.CredentialsActive {
				log.C(ctx).Infof("Storing current credentials for platform %s as old", platform.ID)
				platform.OldCredentials = platform.Credentials
				platform.CredentialsActive = false
			}
			if err := generateCredentials(ctx, platform); err != nil {
				return nil, err
			}
		}
		return h(ctx, platform, labelChanges...)
	}
}

func generateCredentials(ctx context.Context, platform *types.Platform) error {
	credentials, err := types.GenerateCredentials()
	if err != nil {
		log.C(ctx).Error("could not generate credentials for platform")
		return err
	}
	platform.Credentials = credentials
	return nil
}
