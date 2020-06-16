/*
 *    Copyright 2018 The Service Manager Authors
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

package common

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gavv/httpexpect"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type OperationExpectations struct {
	Category          types.OperationCategory
	State             types.OperationState
	ResourceType      types.ObjectType
	Reschedulable     bool
	DeletionScheduled bool
	Error             string
}

type ResourceExpectations struct {
	ID    string
	Type  types.ObjectType
	Ready bool
}

func VerifyResourceExists(smClient *SMExpect, expectations ResourceExpectations) *httpexpect.Object {
	timeoutDuration := 15 * time.Second
	timeout := time.After(timeoutDuration)
	ticker := time.Tick(100 * time.Millisecond)
	for {
		select {
		case <-timeout:
			Fail(fmt.Sprintf("resource with type %s and id %s did not appear in SM after %.0f seconds", expectations.Type, expectations.ID, timeoutDuration.Seconds()))
		case <-ticker:
			resources := smClient.ListWithQuery(expectations.Type.String(), fmt.Sprintf("fieldQuery=id eq '%s'", expectations.ID))
			switch {
			case resources.Length().Raw() == 0:
				By(fmt.Sprintf("Could not find resource with type %s and id %s in SM. Retrying...", expectations.Type, expectations.ID))
			case resources.Length().Raw() > 1:
				Fail(fmt.Sprintf("more than one resource with type %s and id %s was found in SM", expectations.Type, expectations.ID))
			default:
				resourceObject := resources.First().Object()
				readyField := resourceObject.Value("ready").Boolean().Raw()
				if readyField != expectations.Ready {
					Fail(fmt.Sprintf("Expected resource with type %s and id %s to be ready %t but ready was %t", expectations.Type, expectations.ID, expectations.Ready, readyField))
				}
				return resources.First().Object()
			}
		}
	}
}

func VerifyResourceDoesNotExist(smClient *SMExpect, expectations ResourceExpectations) {
	timeoutDuration := 15 * time.Second
	timeout := time.After(timeoutDuration)
	ticker := time.Tick(100 * time.Millisecond)
	for {
		select {
		case <-timeout:
			Fail(fmt.Sprintf("resource with type %s and id %s was still in SM after %.0f seconds", expectations.Type, expectations.ID, timeoutDuration.Seconds()))
		case <-ticker:
			resp := smClient.GET(expectations.Type.String() + "/" + expectations.ID).
				Expect().Raw()
			if resp.StatusCode != http.StatusNotFound {
				By(fmt.Sprintf("Found resource with type %s and id %s but it should be deleted. Retrying...", expectations.Type, expectations.ID))
			} else {
				return
			}
		}
	}
}

func VerifyOperationExists(ctx *TestContext, operationURL string, expectations OperationExpectations) (string, string) {
	timeoutDuration := 15 * time.Second
	timeout := time.After(timeoutDuration)
	ticker := time.Tick(100 * time.Millisecond)

	shouldVerifyWithList := func(expect OperationExpectations) bool {
		if expect.Category == "delete" && expect.State == "succeeded" {
			return false
		}

		if expect.Category == "create" && expect.State == "failed" {
			return false
		}

		return true;
	}

	findLastOperationByList := func(resourceURL, id string) types.Operation {
		resp := ctx.SMWithOAuth.GET(expectations.ResourceType.String()).WithQuery("attach_last_operations", "true").Expect().Status(http.StatusOK).JSON()

		for _, resource := range resp.Path("$.items[*]").Array().Iter() {
			rawRes := resource.Object().Raw()
			if _, ok := rawRes["last_operation"]; ok {
				lastOp := resource.Object().Value("last_operation").Object()
				var resLastOperation types.Operation
				resLastOperation.ID = lastOp.ContainsKey("id").Value("id").String().Raw()
				if resLastOperation.ID == id {
					return resLastOperation
				}
			}
		}
		return types.Operation{}
	}

	for {
		select {
		case <-timeout:
			Fail(fmt.Sprintf("operation matching expectations did not appear in SM after %.0f seconds", timeoutDuration.Seconds()))
		case <-ticker:
			var operation map[string]interface{}
			if len(operationURL) != 0 {
				operation = ctx.SMWithOAuth.GET(operationURL).Expect().Status(http.StatusOK).JSON().Object().Raw()
				category := operation["type"].(string)
				resourceType := operation["resource_type"].(string)
				state := operation["state"].(string)
				id := operation["id"].(string)
				reschedulable := operation["reschedule"].(bool)
				deletionScheduledString := operation["deletion_scheduled"].(string)
				deletionScheduled, err := time.Parse(time.RFC3339Nano, deletionScheduledString)
				if err != nil {
					Fail(fmt.Sprintf("Could not parse time %s into format %s: %s", deletionScheduledString, time.RFC3339Nano, err))
				}
				var foundOp types.Operation
				if shouldVerifyWithList(expectations) {
					foundOp = findLastOperationByList(expectations.ResourceType.String(), id)
				}

				switch {
				case resourceType != string(expectations.ResourceType.String()):
					By(fmt.Sprintf("Found operation with object type %s but expected %s. Continue waiting...", resourceType, expectations.ResourceType))
				case category != string(expectations.Category):
					By(fmt.Sprintf("Found operation with category %s but expected %s. Continue waiting...", category, expectations.Category))
				case state != string(expectations.State):
					By(fmt.Sprintf("Found operation with state %s but expected %s. Continue waiting...", state, expectations.State))
				case reschedulable != expectations.Reschedulable:
					By(fmt.Sprintf("Found operation with reschdulable %t but expected %t. Continue waiting...", reschedulable, expectations.Reschedulable))
				case deletionScheduled.IsZero() == expectations.DeletionScheduled:
					By(fmt.Sprintf("Found operation with deletion schduled %t but expected %t. Continue waiting...", !deletionScheduled.IsZero(), expectations.DeletionScheduled))
				case len(expectations.Error) != 0:
					errs := operation["errors"].(map[string]interface{})
					errMsg := errs["description"].(string)
					if !strings.Contains(errMsg, expectations.Error) {
						By(fmt.Sprintf("Found operation with error %s but expected %s. Continue waiting...", errMsg, expectations.Error))
					} else {
						resourceID := operation["resource_id"].(string)
						By(fmt.Sprintf("Found matching operation with resource_id %s", resourceID))

						return resourceID, operation["id"].(string)
					}
				case shouldVerifyWithList(expectations) && foundOp.ID == "":
					By(fmt.Sprintf("Last operation was not found by listing resource last opereartions when querying for %s, waiting", resourceType))
				default:
					resourceID := operation["resource_id"].(string)
					By(fmt.Sprintf("Found matching operation with resource_id %s", resourceID))
					return resourceID, operation["id"].(string)
				}
			} else {
				By("Operation URL is empty. Searching for operation directly in SMDB...")
				byResourceType := query.ByField(query.EqualsOperator, "resource_type", string(expectations.ResourceType))
				byCategory := query.ByField(query.EqualsOperator, "type", string(expectations.Category))
				byState := query.ByField(query.EqualsOperator, "state", string(expectations.State))
				byReschedulable := query.ByField(query.EqualsOperator, "reschedule", strconv.FormatBool(expectations.Reschedulable))
				orderDesc := query.OrderResultBy("paging_sequence", query.DescOrder)
				objectList, err := ctx.SMRepository.List(context.TODO(), types.OperationType,
					byResourceType, byCategory, byState, byReschedulable, orderDesc)
				if err != nil {
					if err == util.ErrNotFoundInStorage {
						By("operation matching the expectations was not found. Retrying...")
					} else {
						Fail(fmt.Sprintf("could not fetch operation from storage: %s", err))
					}
				} else {
					if objectList.Len() != 0 {
						op := objectList.ItemAt(0).(*types.Operation)

						var foundOp types.Operation
						if shouldVerifyWithList(expectations) {
							foundOp = findLastOperationByList(expectations.ResourceType.String(), op.ID)
						}

						if shouldVerifyWithList(expectations) && foundOp.ID == "" {
							By(fmt.Sprintf("Last operation was not found by listing resource last opereartions when querying for %s, waiting", op.ResourceType))
						} else if op.DeletionScheduled.IsZero() == expectations.DeletionScheduled {
							By("operation matching the expectations was not found. Retrying...")
						} else if expectations.Error != "" && !strings.Contains(string(op.Errors), expectations.Error) {
							By(fmt.Sprintf("Found operation with error %s but expected %s. Continue waiting...", string(op.Errors), expectations.Error))
						} else {
							return op.ResourceID, op.ID
						}
					}
				}
			}
		}
	}
}

type TransitiveResourcesExpectation struct {
	CreatedOfferings     int
	CreatedPlans         int
	CreatedNotifications int
}

func AssertTransitiveResources(operation *types.Operation, expectations TransitiveResourcesExpectation) {
	transitiveResources := operation.TransitiveResources
	actualCreatedOfferings := 0
	actualCreatedPlans := 0
	actualCreatedNotifications := 0

	for _, val := range transitiveResources {
		if val.OperationType == types.CREATE {
			switch val.Type {
			case types.ServiceOfferingType:
				actualCreatedOfferings++
			case types.ServicePlanType:
				actualCreatedPlans++
			case types.NotificationType:
				actualCreatedNotifications++
			}
		}
	}
	Expect(actualCreatedNotifications).To(Equal(expectations.CreatedNotifications))
	Expect(actualCreatedOfferings).To(Equal(expectations.CreatedOfferings))
	Expect(actualCreatedPlans).To(Equal(expectations.CreatedPlans))
}
