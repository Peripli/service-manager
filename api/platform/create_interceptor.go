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

package platform

import (
	"context"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/extension"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/types"
)

const (
	platformCreateInterceptorProviderName = "create-platform"
)

type createInterceptorProvider struct {
	encrypter security.Encrypter
}

func (c *createInterceptorProvider) Provide() extension.CreateInterceptor {
	return &CreateInterceptor{
		Encrypter: c.encrypter,
	}
}
func (c *createInterceptorProvider) Name() string {
	return platformCreateInterceptorProviderName
}

type CreateInterceptor struct {
	Encrypter security.Encrypter
}

func (c *CreateInterceptor) OnAPICreate(h extension.InterceptCreateOnAPI) extension.InterceptCreateOnAPI {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		credentials, err := types.GenerateCredentials()
		if err != nil {
			log.C(ctx).Error("Could not generate credentials for platform")
			return nil, err
		}
		plainPassword := credentials.Basic.Password
		transformedPassword, err := c.Encrypter.Encrypt(ctx, []byte(plainPassword))
		if err != nil {
			return nil, err
		}
		credentials.Basic.Password = string(transformedPassword)
		(obj.(types.Secured)).SetCredentials(credentials)

		object, err := h(ctx, obj)
		if err != nil {
			return nil, err
		}

		credentials.Basic.Password = plainPassword
		(object.(types.Secured)).SetCredentials(credentials)
		return object, nil
	}

}

func (*CreateInterceptor) OnTxCreate(f extension.InterceptCreateOnTx) extension.InterceptCreateOnTx {
	return f
}
