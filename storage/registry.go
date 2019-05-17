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

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/log"
)

func OpenWithSafeTermination(ctx context.Context, s Storage, options *Settings, wg *sync.WaitGroup) error {
	if s == nil || options == nil {
		return fmt.Errorf("storage and storage settings cannot be nil")
	}

	util.StartInWaitGroupWithContext(ctx, func(c context.Context) {
		<-c.Done()
		log.C(c).Debug("Context cancelled. Closing storage...")
		if err := s.Close(); err != nil {
			log.D().Error(err)
		}
	}, wg)

	if err := s.Open(options); err != nil {
		return fmt.Errorf("error opening storage: %s", err)
	}

	return nil
}
