/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package storage

import (
	"context"
	"fmt"
	"sync"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
)

func InitializeWithSafeTermination(ctx context.Context, s Storage, settings *Settings, wg *sync.WaitGroup, decorators ...TransactionalRepositoryDecorator) (TransactionalRepository, error) {
	if s == nil || settings == nil {
		return nil, fmt.Errorf("storage and storage settings cannot be nil")
	}

	util.StartInWaitGroupWithContext(ctx, func(c context.Context) {
		<-c.Done()
		log.C(c).Debug("Context cancelled. Closing storage...")
		if err := s.Close(); err != nil {
			log.C(c).Error(err)
		}
	}, wg)

	if err := s.Open(settings); err != nil {
		return nil, fmt.Errorf("error opening storage: %s", err)
	}

	var decoratedRepository TransactionalRepository
	var err error
	decoratedRepository = s
	for i := range decorators {
		decoratedRepository, err = decorators[len(decorators)-1-i](decoratedRepository)
		if err != nil {
			return nil, err
		}
	}

	return decoratedRepository, nil
}
