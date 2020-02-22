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

package api

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/gofrs/uuid"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

// CredentialsController implements api.Controller by providing logic for broker platform credential storage/update
type CredentialsController struct {
	*BaseController
}

func NewCredentialsController(ctx context.Context, options *Options) *CredentialsController {
	return &CredentialsController{
		BaseController: NewController(ctx, options, web.BrokerPlatformCredentialsURL, types.BrokerPlatformCredentialType, func() types.Object {
			return &types.BrokerPlatformCredential{}
		}),
	}
}

func (c *CredentialsController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPost,
				Path:   c.resourceBaseURL,
			},
			Handler: c.registerCredentials,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPatch,
				Path:   c.resourceBaseURL,
			},
			Handler: c.updateCredentials,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   c.resourceBaseURL,
			},
			Handler: c.deleteCredentials,
		},
	}
}

func (c *BaseController) registerCredentials(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Creating new broker platform credentials")

	platform, err := osb.ExtractPlatformFromContext(ctx)
	if err != nil {
		return nil, err
	}

	credentials := &types.BrokerPlatformCredential{}
	if err := util.BytesToObject(r.Body, credentials); err != nil {
		return nil, err
	}

	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for %s: %s", c.objectType, err)
	}
	credentials.SetID(UUID.String())

	currentTime := time.Now().UTC()
	credentials.SetCreatedAt(currentTime)
	credentials.SetUpdatedAt(currentTime)
	credentials.SetReady(true)

	credentials.PlatformID = platform.ID
	credentials.OldUsername = ""
	credentials.OldPasswordHash = ""

	createdObj, err := c.repository.Create(ctx, credentials)
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	return util.NewJSONResponse(http.StatusCreated, createdObj)
}

func (c *BaseController) updateCredentials(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Updating broker platform credentials")

	platform, err := osb.ExtractPlatformFromContext(ctx)
	if err != nil {
		return nil, err
	}

	credentials := &types.BrokerPlatformCredential{}
	if err := util.BytesToObject(r.Body, credentials); err != nil {
		return nil, err
	}

	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", platform.ID),
		query.ByField(query.EqualsOperator, "broker_id", credentials.BrokerID),
	}
	objFromDB, err := c.repository.Get(ctx, c.objectType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	brokerPlatformCredentials := objFromDB.(*types.BrokerPlatformCredential)

	brokerPlatformCredentials.OldUsername = brokerPlatformCredentials.Username
	brokerPlatformCredentials.OldPasswordHash = brokerPlatformCredentials.PasswordHash

	brokerPlatformCredentials.Username = credentials.Username
	brokerPlatformCredentials.PasswordHash = credentials.PasswordHash

	object, err := c.repository.Update(ctx, brokerPlatformCredentials, query.LabelChanges{})
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	return util.NewJSONResponse(http.StatusOK, object)
}

func (c *BaseController) deleteCredentials(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Delete broker platform credentials")

	platform, err := osb.ExtractPlatformFromContext(ctx)
	if err != nil {
		return nil, err
	}

	credentials := &types.BrokerPlatformCredential{}
	if err := util.BytesToObject(r.Body, credentials); err != nil {
		return nil, err
	}

	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", platform.ID),
		query.ByField(query.EqualsOperator, "broker_id", credentials.BrokerID),
	}

	if err := c.repository.Delete(ctx, types.BrokerPlatformCredentialType, criteria...); err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}
