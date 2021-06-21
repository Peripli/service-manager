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
	"net/http"
	"time"

	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	"github.com/tidwall/gjson"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

// credentialsController implements api.Controller by providing logic for broker platform credential storage/update
type credentialsController struct {
	repository storage.Repository
}

// Routes provides endpoints for rotating broker credentials for particular platform
func (c *credentialsController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPut,
				Path:   web.BrokerPlatformCredentialsURL,
			},
			Handler: c.setCredentials,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPut,
				Path:   fmt.Sprintf("%s/{%s}/activate", web.BrokerPlatformCredentialsURL, web.PathParamResourceID),
			},
			Handler: c.activateCredentials,
		},
	}
}
func (c *credentialsController) setCredentials(r *web.Request) (*web.Response, error) {
	ctx := r.Context()

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
		log.C(ctx).Infof("Notification with id %s is found in broker platform request body, fetching from DB...", body.NotificationID)
		criteria := []query.Criterion{
			query.ByField(query.EqualsOperator, "id", body.NotificationID),
			query.ByField(query.EqualsOrNilOperator, "platform_id", body.PlatformID),
			query.ByField(query.EqualsOperator, "resource", string(types.ServiceBrokerType)),
		}

		obj, err := c.repository.Get(ctx, types.NotificationType, criteria...)
		if err != nil {
			if err == util.ErrNotFoundInStorage {
				log.C(ctx).Errorf("Notification from broker platform credentials request with id %s not found in DB", body.NotificationID)
				return nil, &util.HTTPError{
					ErrorType:   "CredentialsError",
					Description: fmt.Sprintf("Invalid notification ID %s - request rejected", body.NotificationID),
					StatusCode:  http.StatusBadRequest,
				}
			}
			return nil, util.HandleStorageError(err, types.BrokerPlatformCredentialType.String())
		}

		notification := obj.(*types.Notification)

		if gjson.GetBytes(notification.Payload, "new.resource.id").String() != body.BrokerID {
			log.C(ctx).Errorf("Notification from broker platform credentials request with id %s is for different broker", body.NotificationID)
			return nil, &util.HTTPError{
				ErrorType:   "CredentialsError",
				Description: fmt.Sprintf("Invalid notification ID %s - request rejected", body.NotificationID),
				StatusCode:  http.StatusBadRequest,
			}
		}
		log.C(ctx).Infof("Notification from broker platform credentials request with id %s found in DB", body.NotificationID)
	}

	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "platform_id", platform.ID),
		query.ByField(query.EqualsOperator, "broker_id", body.BrokerID),
	}
	objFromDB, err := c.repository.Get(ctx, types.BrokerPlatformCredentialType, criteria...)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return c.registerCredentials(ctx, body)
		}
		return nil, util.HandleStorageError(err, types.BrokerPlatformCredentialType.String())
	}

	if body.NotificationID == "" {
		log.C(ctx).Error("Notification id not provided and broker platform credentials already exists...")
		return nil, &util.HTTPError{
			ErrorType:   "CredentialsError",
			Description: fmt.Sprint("Invalid request - cannot update existing credentials"),
			StatusCode:  http.StatusConflict,
		}
	}

	credentialsFromDB := objFromDB.(*types.BrokerPlatformCredential)
	return c.updateCredentials(ctx, body, credentialsFromDB)
}

func (c *credentialsController) registerCredentials(ctx context.Context, credentials *types.BrokerPlatformCredential) (*web.Response, error) {
	log.C(ctx).Info("Creating new broker platform credentials")

	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for %s: %s", types.BrokerPlatformCredentialType.String(), err)
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
		return nil, util.HandleStorageError(err, types.BrokerPlatformCredentialType.String())
	}
	log.C(ctx).Infof("Successfully created credentials for platform %s and broker %s", credentials.PlatformID, credentials.BrokerID)

	return util.NewJSONResponse(http.StatusOK, createdObj)
}

func (c *credentialsController) updateCredentials(ctx context.Context, body, credentialsFromDB *types.BrokerPlatformCredential) (*web.Response, error) {
	log.C(ctx).Debugf("Updating broker platform credentials")

	if credentialsFromDB.Active || len(credentialsFromDB.OldUsername) == 0 || len(credentialsFromDB.OldPasswordHash) == 0 {
		log.C(ctx).Debug("Updating old username and old password")
		credentialsFromDB.OldUsername = credentialsFromDB.Username
		credentialsFromDB.OldPasswordHash = credentialsFromDB.PasswordHash
	} else {
		log.C(ctx).Info("Current credentials were not active, will not be saved to old username and old password")
	}

	credentialsFromDB.Username = body.Username
	credentialsFromDB.PasswordHash = body.PasswordHash
	credentialsFromDB.Active = false

	object, err := c.repository.Update(ctx, credentialsFromDB, types.LabelChanges{})
	if err != nil {
		return nil, util.HandleStorageError(err, types.BrokerPlatformCredentialType.String())
	}
	log.C(ctx).Infof("Successfully rotated credentials for platform %s and broker %s", credentialsFromDB.PlatformID, credentialsFromDB.BrokerID)

	return util.NewJSONResponse(http.StatusOK, object)
}

func (c *credentialsController) activateCredentials(r *web.Request) (*web.Response, error) {
	ctx := r.Context()

	byID := query.ByField(query.EqualsOperator, "id", r.PathParams[web.PathParamResourceID])
	objFromDB, err := c.repository.Get(ctx, types.BrokerPlatformCredentialType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.BrokerPlatformCredentialType.String())
	}

	credentialsFromDB := objFromDB.(*types.BrokerPlatformCredential)
	credentialsFromDB.Active = true
	credentialsFromDB.OldPasswordHash = ""
	credentialsFromDB.OldUsername = ""
	object, err := c.repository.Update(ctx, credentialsFromDB, types.LabelChanges{})
	if err != nil {
		return nil, util.HandleStorageError(err, types.BrokerPlatformCredentialType.String())
	}
	log.C(ctx).Infof("Successfully activated credentials for platform %s and broker %s", credentialsFromDB.PlatformID, credentialsFromDB.BrokerID)

	return util.NewJSONResponse(http.StatusOK, object)
}
