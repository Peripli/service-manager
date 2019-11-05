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

package storage

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

// NotificationCleaner schedules a go routine which cleans old notifications
type NotificationCleaner struct {
	started bool

	Storage  Repository
	Settings Settings
}

// Start schedules the cleaner. It cannot be used concurrently.
func (nc *NotificationCleaner) Start(ctx context.Context, group *sync.WaitGroup) error {
	if nc.started {
		return errors.New("notification cleaner already started")
	}
	nc.started = true
	group.Add(1)
	go func() {
		defer func() {
			nc.started = false
			group.Done()
		}()
		cleanInterval := nc.Settings.Notification.CleanInterval
		log.C(ctx).Infof("Scheduling notification cleaning every %s", cleanInterval.String())
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(cleanInterval):
				nc.clean(ctx)
			}
		}
	}()
	return nil
}

func (nc *NotificationCleaner) clean(ctx context.Context) {
	cleanTimestamp := util.ToRFCNanoFormat(time.Now().Add(-nc.Settings.Notification.KeepFor))
	log.C(ctx).Infof("Deleting notifications created before %s", cleanTimestamp)

	q := query.ByField(query.LessThanOperator, "created_at", cleanTimestamp)
	deletedNotifications, err := nc.Storage.Delete(ctx, types.NotificationType, q)

	if err == util.ErrNotFoundInStorage {
		log.C(ctx).Debug("no old notifications to delete")
	} else if err != nil {
		log.C(ctx).WithError(err).Error("could not delete old notifications")
	} else {
		log.C(ctx).Infof("successfully deleted %d old notifications", deletedNotifications.Len())
	}
}
