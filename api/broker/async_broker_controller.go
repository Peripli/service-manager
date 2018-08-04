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

package broker

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/work"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

func (c *Controller) asyncCreateBroker(request *web.Request) (*web.Response, error) {
	logrus.Debug("Creating new broker")
	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for broker: %s", err)
	}

	job := work.Job{
		EntityType: work.EntityBroker,
		Action:     work.ActionCreate,
		EntityId:   UUID.String(),
		Data:       request.Body,
	}
	responseBody := fmt.Sprintf(`{"id": "%s"}`, job.EntityId)
	c.JobQueue <- job
	return &web.Response{
		StatusCode: http.StatusOK,
		Body:       []byte(responseBody),
	}, nil
}

func (c *Controller) asyncDeleteBroker(request *web.Request) (*web.Response, error) {
	brokerID := request.PathParams[reqBrokerID]
	logrus.Debugf("Deleting broker with id %s", brokerID)
	job := work.Job{
		EntityType: work.EntityBroker,
		Action:     work.ActionDelete,
		EntityId:   brokerID,
	}
	c.JobQueue <- job
	return &web.Response{
		StatusCode: http.StatusOK,
		Body:       []byte(`{}`),
	}, nil
}
