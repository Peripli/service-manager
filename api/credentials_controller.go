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
	"github.com/tidwall/gjson"
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
				Method: http.MethodPut,
				Path:   c.resourceBaseURL,
			},
			Handler: c.setCredentials,
		},
	}
}
func (c *BaseController) setCredentials(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debugf("Creating new broker platform credentials")

	platform, err := osb.ExtractPlatformFromContext(ctx)
	if err != nil {
		return nil, err
	}

	body := &types.BrokerPlatformCredential{}
	if err := util.BytesToObject(r.Body, body); err != nil {
		return nil, err
	}
	body.PlatformID = platform.ID

	if body.NotificationID != "" {
		criteria := []query.Criterion{
			query.ByField(query.EqualsOperator, "id", body.NotificationID),
			query.ByField(query.EqualsOrNilOperator, "platform_id", body.PlatformID),
			query.ByField(query.EqualsOperator, "resource", string(types.ServiceBrokerType)),
		}

		obj, err := c.repository.Get(ctx, types.NotificationType, criteria...)
		if err != nil {
			if err == util.ErrNotFoundInStorage {
				return nil, &util.HTTPError{
					ErrorType:   "CredentialsError",
					Description: fmt.Sprintf("Invalid notification ID %s - request rejected", body.NotificationID),
					StatusCode:  http.StatusConflict,
				}
			}
			return nil, util.HandleStorageError(err, c.objectType.String())
		}

		notification := obj.(*types.Notification)

		if gjson.GetBytes(notification.Payload, "new.resource.id").String() != body.BrokerID {
			return nil, &util.HTTPError{
				ErrorType:   "CredentialsError",
				Description: fmt.Sprintf("Invalid notification ID %s - request rejected", body.NotificationID),
				StatusCode:  http.StatusConflict,
			}
		}
	}

	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", platform.ID),
		query.ByField(query.EqualsOperator, "broker_id", body.BrokerID),
	}
	objFromDB, err := c.repository.Get(ctx, c.objectType, criteria...)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return c.registerCredentials(ctx, body)
		}
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	if body.NotificationID == "" {
		return nil, &util.HTTPError{
			ErrorType:   "CredentialsError",
			Description: fmt.Sprint("Invalid request - cannot update existing credentials"),
			StatusCode:  http.StatusConflict,
		}
	}

	credentialsFromDB := objFromDB.(*types.BrokerPlatformCredential)
	return c.updateCredentials(ctx, body, credentialsFromDB)
}

func (c *BaseController) registerCredentials(ctx context.Context, credentials *types.BrokerPlatformCredential) (*web.Response, error) {
	log.C(ctx).Debugf("Creating new broker platform credentials")

	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for %s: %s", c.objectType, err)
	}
	credentials.SetID(UUID.String())

	currentTime := time.Now().UTC()
	credentials.SetCreatedAt(currentTime)
	credentials.SetUpdatedAt(currentTime)
	credentials.SetReady(true)

	credentials.OldUsername = ""
	credentials.OldPasswordHash = ""

	createdObj, err := c.repository.Create(ctx, credentials)
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	return util.NewJSONResponse(http.StatusOK, createdObj)
}

func (c *BaseController) updateCredentials(ctx context.Context, body, credentialsFromDB *types.BrokerPlatformCredential) (*web.Response, error) {
	log.C(ctx).Debugf("Updating broker platform credentials")

	credentialsFromDB.OldUsername = credentialsFromDB.Username
	credentialsFromDB.OldPasswordHash = credentialsFromDB.PasswordHash

	credentialsFromDB.Username = body.Username
	credentialsFromDB.PasswordHash = body.PasswordHash

	object, err := c.repository.Update(ctx, credentialsFromDB, types.LabelChanges{})
	if err != nil {
		return nil, util.HandleStorageError(err, c.objectType.String())
	}

	return util.NewJSONResponse(http.StatusOK, object)
}
