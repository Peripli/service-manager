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

package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

// notificationStorage storage for getting notifications and last revision
//go:generate counterfeiter . notificationStorage
type notificationStorage interface {
	GetNotification(ctx context.Context, id string) (*types.Notification, error)

	GetNotificationByRevision(ctx context.Context, revision int64) (*types.Notification, error)

	ListNotifications(ctx context.Context, platformID string, from, to int64) ([]*types.Notification, error)

	// GetLastRevision returns the last received notification revision
	GetLastRevision(ctx context.Context) (int64, error)
}

// NewNotificationStorage returns new notification storage
func NewNotificationStorage(st storage.Storage) (*notificationStorageImpl, error) {
	pgStorage, ok := st.(*Storage)
	if !ok {
		return nil, errors.New("expected notification storage to be Postgres")
	}
	return &notificationStorageImpl{
		storage: pgStorage,
	}, nil
}

type notificationStorageImpl struct {
	storage *Storage
}

func (ns *notificationStorageImpl) GetLastRevision(ctx context.Context) (int64, error) {
	result := make([]*Notification, 0, 1)
	sqlString := fmt.Sprintf("SELECT revision FROM %s ORDER BY revision DESC LIMIT 1", NotificationTable)
	err := ns.storage.SelectContext(ctx, &result, sqlString)
	if err != nil {
		return 0, fmt.Errorf("could not get last notification revision from db %v", err)
	}
	if len(result) == 0 {
		return types.InvalidRevision, nil
	}
	return result[0].Revision, nil
}

func (ns *notificationStorageImpl) GetNotification(ctx context.Context, id string) (*types.Notification, error) {
	byID := query.ByField(query.EqualsOperator, "id", id)
	notificationObj, err := ns.storage.Get(ctx, types.NotificationType, byID)
	if err != nil {
		return nil, err
	}
	return notificationObj.(*types.Notification), nil
}

func (ns *notificationStorageImpl) ListNotifications(ctx context.Context, platformID string, from, to int64) ([]*types.Notification, error) {
	listQuery1 := query.ByField(query.GreaterThanOperator, "revision", strconv.FormatInt(from, 10))
	listQuery2 := query.ByField(query.LessThanOrEqualOperator, "revision", strconv.FormatInt(to, 10))
	filterByPlatform := query.ByField(query.EqualsOrNilOperator, "platform_id", platformID)
	orderByRevision := query.OrderResultBy("revision", query.AscOrder)
	objectList, err := ns.storage.List(ctx, types.NotificationType, orderByRevision, listQuery1, listQuery2, filterByPlatform)
	if err != nil {
		return nil, err
	}
	notificationsList := objectList.(*types.Notifications)

	return notificationsList.Notifications, nil
}

func (ns *notificationStorageImpl) GetNotificationByRevision(ctx context.Context, revision int64) (*types.Notification, error) {
	revisionQuery := query.ByField(query.EqualsOperator, "revision", strconv.FormatInt(revision, 10))
	objectList, err := ns.storage.List(ctx, types.NotificationType, revisionQuery)
	if err != nil {
		return nil, err
	}
	if objectList.Len() == 0 {
		return nil, util.ErrNotFoundInStorage
	}
	if objectList.Len() > 1 {
		return nil, fmt.Errorf("expected one notification with revision %d got %d", revision, objectList.Len())
	}
	notificationsList := objectList.(*types.Notifications)
	return notificationsList.Notifications[0], nil
}
