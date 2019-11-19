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

package reconcile_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/agent"

	"github.com/Peripli/service-manager/pkg/agent/reconcile"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	fakeSMAppHost        = "https://sm.com"
	fakeProxyAppHost     = "https://smproxy.com"
	fakeProxyPathPattern = fakeProxyAppHost + agent.APIPrefix + "/%s"
)

func TestReconcile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reconcile Suite")
}

func brokerProxyName(brokerName, brokerID string) string {
	return fmt.Sprintf("%s%s-%s", reconcile.DefaultProxyBrokerPrefix, brokerName, brokerID)
}

func verifyInvocationsUseSameCorrelationID(invocations []map[string][][]interface{}) {
	var expectedCorrelationID string
	// for each client
	for _, client := range invocations {
		// for each method type
		for _, methodInvocation := range client {
			// for each time this method is used
			for _, args := range methodInvocation {
				// if there were args in the call
				if len(args) != 0 {
					// the first arg of each client method invocation is the context and it contains the correlation ID
					ctx := args[0].(context.Context)
					if expectedCorrelationID == "" {
						// store the first found correlation ID in a variable
						expectedCorrelationID = log.CorrelationIDFromContext(ctx)
					} else {
						// verify that every call done by every client contains the same correlation ID value in the provided context
						Expect(log.CorrelationIDFromContext(ctx)).To(Equal(expectedCorrelationID))
					}
				}
			}
		}
	}
}
