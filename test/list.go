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

package test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/gofrs/uuid"
	"net/url"
	"strings"
	"time"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/gavv/httpexpect"

	. "github.com/onsi/ginkgo/extensions/table"

	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test/common"
)

type listOpEntry struct {
	resourcesToExpectBeforeOp []common.Object

	queryTemplate               string
	queryArgs                   common.Object
	resourcesToExpectAfterOp    []common.Object
	resourcesNotToExpectAfterOp []common.Object
	expectedStatusCode          int
}

func DescribeListTestsFor(ctx *common.TestContext, t TestCase, responseMode ResponseMode) bool {
	var r []common.Object
	var rWithMandatoryFields common.Object
	commonLabelKey := "labelKey1"
	commonLabelValue := "1"

	attachLabel := func(obj common.Object) common.Object {
		patchLabels := []*types.LabelChange{
			{
				Operation: types.AddLabelOperation,
				Key:       commonLabelKey,
				Values:    []string{commonLabelValue},
			},
			{
				Operation: types.AddLabelOperation,
				Key:       "labelKey2",
				Values:    []string{"str"},
			},
			{
				Operation: types.AddLabelOperation,
				Key:       "labelKey3",
				Values:    []string{`{"key1": "val1", "key2": "val2"}`},
			},
		}

		t.PatchResource(ctx, t.StrictlyTenantScoped, t.API, obj["id"].(string), types.ObjectType(t.API), patchLabels, bool(responseMode))
		var result *httpexpect.Object
		if t.StrictlyTenantScoped {
			result = ctx.SMWithOAuthForTenant.ListWithQuery(t.API, fmt.Sprintf("fieldQuery=id eq '%s'", obj["id"].(string))).First().Object()
		} else {
			result = ctx.SMWithOAuth.ListWithQuery(t.API, fmt.Sprintf("fieldQuery=id eq '%s'", obj["id"].(string))).First().Object()
		}
		result.ContainsKey("labels")
		resultObject := result.Raw()
		delete(resultObject, "credentials")

		return resultObject
	}

	By(fmt.Sprintf("Attempting to create a random resource of %s with mandatory fields only", t.API))
	rWithMandatoryFields = t.ResourceWithoutNullableFieldsBlueprint(ctx, ctx.SMWithOAuth, bool(responseMode))
	for i := 0; i < 5; i++ {
		By(fmt.Sprintf("Attempting to create a random resource of %s", t.API))

		gen := t.ResourceBlueprint(ctx, ctx.SMWithOAuth, bool(responseMode))
		gen = attachLabel(gen)
		stripObject(gen, t.ResourcePropertiesToIgnore...)
		r = append(r, gen)
	}

	entries := []TableEntry{
		Entry("return 200 when contains operator is valid",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%s contains '%v'",
				queryArgs:                 common.RemoveBooleanArgs(common.RemoveNumericAndDateArgs(r[0])),
				resourcesToExpectAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("return 400 when contains operator is applied on non-strings fields",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%s contains '%v'",
				queryArgs:                 common.RemoveNonNumericOrDateArgs(common.CopyFields(r[0])),
				resourcesToExpectAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0]},
				queryTemplate:             "%s eq '%v'",
				queryArgs:                 r[0],
				resourcesToExpectAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp:   []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:               "%s ne '%v'",
				queryArgs:                   r[0],
				resourcesNotToExpectAfterOp: []common.Object{r[0]},
				expectedStatusCode:          http.StatusOK,
			},
		),

		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%[1]s in ('%[2]v','%[2]v','%[2]v')",
				queryArgs:                 r[0],
				resourcesToExpectAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),

		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%s in ('%v')",
				queryArgs:                 r[0],
				resourcesToExpectAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp:   []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:               "%[1]s notin ('%[2]v','%[2]v','%[2]v')",
				queryArgs:                   r[0],
				resourcesNotToExpectAfterOp: []common.Object{r[0]},
				expectedStatusCode:          http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp:   []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:               "%s notin ('%v')",
				queryArgs:                   r[0],
				resourcesNotToExpectAfterOp: []common.Object{r[0]},
				expectedStatusCode:          http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp:   []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:               "%s gt '%v'",
				queryArgs:                   common.RemoveNonNumericOrDateArgs(r[0]),
				resourcesNotToExpectAfterOp: []common.Object{r[0]},
				expectedStatusCode:          http.StatusOK,
			},
		),
		Entry("returns 200 for greater than or equal queries",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%s ge %v",
				queryArgs:                 common.RemoveNonNumericOrDateArgs(r[0]),
				resourcesToExpectAfterOp:  []common.Object{r[0], r[1], r[2], r[3]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 400 for greater than or equal queries when query args are non numeric or date",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%s ge %v",
				queryArgs:                 common.RemoveNumericAndDateArgs(r[0]),
				resourcesToExpectAfterOp:  []common.Object{r[0], r[1], r[2], r[3]},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 200 for less than or equal queries",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%s le %v",
				queryArgs:                 common.RemoveNonNumericOrDateArgs(r[0]),
				resourcesToExpectAfterOp:  []common.Object{r[0], r[1], r[2], r[3]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 400 for less than or equal queries when query args are non numeric ор дате",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%s le %v",
				queryArgs:                 common.RemoveNumericAndDateArgs(r[0]),
				resourcesToExpectAfterOp:  []common.Object{r[0], r[1], r[2], r[3]},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp:   []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:               "%s lt '%v'",
				queryArgs:                   common.RemoveNonNumericOrDateArgs(r[0]),
				resourcesNotToExpectAfterOp: []common.Object{r[0]},
				expectedStatusCode:          http.StatusOK,
			},
		),
		Entry("returns 200 for field queries",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], rWithMandatoryFields},
				queryTemplate:             "%s en '%v'",
				queryArgs:                 common.RemoveNotNullableFieldAndLabels(r[0], rWithMandatoryFields),
				resourcesToExpectAfterOp:  []common.Object{r[0], rWithMandatoryFields},
				expectedStatusCode:        http.StatusOK,
			},
		),

		Entry("returns 400 for label queries with operator en",
			listOpEntry{
				queryTemplate: "%s en '%v'",
				queryArgs: common.Object{
					"labels": map[string]interface{}{
						commonLabelKey: []interface{}{
							"str",
						},
					}},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 200 for JSON fields with stripped new lines",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0]},
				queryTemplate:             "%s eq '%v'",
				queryArgs:                 common.RemoveNonJSONArgs(r[0]),
				resourcesToExpectAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 400 when query operator is invalid",
			listOpEntry{
				queryTemplate:      "%s @@ '%v'",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when label query is duplicated",
			listOpEntry{
				queryTemplate: "%[1]s eq '%[2]v' and %[1]s and '%[2]v'",
				queryArgs: common.Object{
					"labels": common.CopyLabels(r[0]),
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when operator is not properly separated with right space from operands",
			listOpEntry{
				queryTemplate:      "%s eq'%v'",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 200 when field query is duplicated",
			listOpEntry{
				queryTemplate:      "%[1]s eq '%[2]v' and %[1]s eq '%[2]v'",
				queryArgs:          common.CopyFields(r[0]),
				expectedStatusCode: http.StatusOK,
			},
		),
		Entry("returns 400 when operator is not properly separated with left space from operands",
			listOpEntry{
				queryTemplate:      "%seq '%v'",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when field query left operands are unknown",
			listOpEntry{
				queryTemplate:      "%[1]s in ('%[2]v', '%[2]v')",
				queryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 200 when label query left operands are unknown",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%[1]s in ('%[2]v','%[2]v')",
				queryArgs: common.Object{
					"labels": map[string]interface{}{
						"unknown": []interface{}{
							"unknown",
						},
					}},
				resourcesNotToExpectAfterOp: []common.Object{r[0], r[1], r[2], r[3]},
				expectedStatusCode:          http.StatusOK,
			},
		),
		Entry("returns 400 when single value operator is used with multiple right value arguments",
			listOpEntry{
				queryTemplate:      "%[1]s ne ('%[2]v','%[2]v','%[2]v')",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when numeric operator is used with non-numeric operands",
			listOpEntry{
				queryTemplate:      "%s < '%v'",
				queryArgs:          common.RemoveNumericAndDateArgs(r[0]),
				expectedStatusCode: http.StatusBadRequest,
			},
		),
	}

	verifyListOpWithAuth := func(listOpEntry listOpEntry, query string, auth *common.SMExpect) {
		var expectedAfterOpIDs []string
		var unexpectedAfterOpIDs []string
		expectedAfterOpIDs = common.ExtractResourceIDs(listOpEntry.resourcesToExpectAfterOp)
		unexpectedAfterOpIDs = common.ExtractResourceIDs(listOpEntry.resourcesNotToExpectAfterOp)

		By(fmt.Sprintf("[TEST]: Verifying expected %s before operation after present", t.API))
		beforeOpArray := auth.List(t.API)

		for _, v := range beforeOpArray.Iter() {
			obj := v.Object().Raw()
			stripObject(obj, t.ResourcePropertiesToIgnore...)
		}

		for _, entity := range listOpEntry.resourcesToExpectBeforeOp {
			stripObject(entity, t.ResourcePropertiesToIgnore...)
			beforeOpArray.Contains(entity)
		}

		By("[TEST]: ======= Expectations Summary =======")

		By(fmt.Sprintf("[TEST]: Listing %s with %s", t.API, query))
		By(fmt.Sprintf("[TEST]: Currently present resources: '%v'", r))
		By(fmt.Sprintf("[TEST]: Expected %s ids after operations: %s", t.API, expectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Unexpected %s ids after operations: %s", t.API, unexpectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Expected status code %d", listOpEntry.expectedStatusCode))

		By("[TEST]: ====================================")

		By(fmt.Sprintf("[TEST]: Verifying expected status code %d is returned from list operation", listOpEntry.expectedStatusCode))

		if listOpEntry.expectedStatusCode != http.StatusOK {
			By("[TEST]: Verifying error and description fields are returned after list operation")
			req := auth.GET(t.API)
			if query != "" {
				req = req.WithQueryString(query)
			}
			req.Expect().Status(listOpEntry.expectedStatusCode).JSON().Object().Keys().Contains("error", "description")
		} else {
			array := auth.ListWithQuery(t.API, query)
			for _, v := range array.Iter() {
				obj := v.Object().Raw()
				stripObject(obj, t.ResourcePropertiesToIgnore...)
			}

			if listOpEntry.resourcesToExpectAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying expected %s are returned after list operation", t.API))
				for _, entity := range listOpEntry.resourcesToExpectAfterOp {
					Expect(entity["ready"].(bool)).To(BeTrue())
					stripObject(entity, t.ResourcePropertiesToIgnore...)
					array.Contains(entity)
				}
			}

			if listOpEntry.resourcesNotToExpectAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying unexpected %s are NOT returned after list operation", t.API))

				for _, entity := range listOpEntry.resourcesNotToExpectAfterOp {
					stripObject(entity, t.ResourcePropertiesToIgnore...)
					array.NotContains(entity)
				}
			}
		}
	}

	verifyListOp := func(listOpEntry listOpEntry, query string) {
		verifyListOpWithAuth(listOpEntry, query, ctx.SMWithOAuth)
	}

	return Describe("List", func() {
		Context("with basic auth", func() {
			if !t.DisableBasicAuth {
				It("returns 200", func() {
					ctx.SMWithBasic.GET(t.API).
						Expect().
						Status(http.StatusOK)
				})
			}
		})

		Context("by date", func() {
			It("returns 200 when date is properly formatted", func() {
				createdAtValue := ctx.SMWithOAuth.ListWithQuery(t.API, fmt.Sprintf("fieldQuery=id eq '%s'", r[0]["id"].(string))).First().Object().Value("created_at").String().Raw()
				parsed, err := time.Parse(time.RFC3339Nano, createdAtValue)
				Expect(err).ToNot(HaveOccurred())
				location, err := time.LoadLocation("America/New_York")
				Expect(err).ToNot(HaveOccurred())
				timeInZone := parsed.In(location)
				offsetCreatedAtValue := timeInZone.Format(time.RFC3339Nano)
				escapedCreatedAtValue := url.QueryEscape(offsetCreatedAtValue)
				ctx.SMWithOAuth.ListWithQuery(t.API, fmt.Sprintf("fieldQuery=%s eq %s", "created_at", escapedCreatedAtValue)).
					Element(0).Object().Value("id").Equal(r[0]["id"])
			})
		})

		Context("when query contains special symbols", func() {
			var obj common.Object
			labelKey := commonLabelKey
			labelValue := "symbols!that@are#url$encoded%when^making a*request("
			BeforeEach(func() {
				obj = t.ResourceBlueprint(ctx, ctx.SMWithOAuth, bool(responseMode))
				patchLabels := []*types.LabelChange{
					{
						Operation: types.AddLabelOperation,
						Key:       labelKey,
						Values:    []string{labelValue},
					},
				}
				t.PatchResource(ctx, t.StrictlyTenantScoped, t.API, obj["id"].(string), types.ObjectType(t.API), patchLabels, bool(responseMode))
			})

			It("returns 200", func() {
				ctx.SMWithOAuth.ListWithQuery(t.API, fmt.Sprintf("labelQuery=%s eq '%s'", labelKey, url.QueryEscape(labelValue))).
					Path("$[*].id").Array().Contains(obj["id"])
			})
		})

		Context("with bearer auth", func() {
			if !t.DisableTenantResources {
				Context("when authenticating with tenant scoped token", func() {
					const resourceSpecificLabel = "resourceSpecificLabel"
					var rForTenant common.Object

					BeforeEach(func() {
						rForTenant = t.ResourceBlueprint(ctx, ctx.SMWithOAuthForTenant, bool(responseMode))
						patchLabels := []*types.LabelChange{
							{
								Operation: types.AddLabelOperation,
								Key:       commonLabelKey,
								Values:    []string{commonLabelValue},
							},
							{
								Operation: types.AddLabelOperation,
								Key:       resourceSpecificLabel,
								Values:    []string{commonLabelValue},
							},
						}
						resourceID := rForTenant["id"].(string)
						t.PatchResource(ctx, t.StrictlyTenantScoped, t.API, resourceID, types.ObjectType(t.API), patchLabels, bool(responseMode))

						rForTenant = ctx.SMWithOAuth.ListWithQuery(t.API, fmt.Sprintf("fieldQuery=id eq '%s'", resourceID)).First().Object().Raw()
					})

					It("returns only resources with specific label", func() {
						verifyListOpWithAuth(listOpEntry{
							resourcesToExpectAfterOp:    []common.Object{rForTenant},
							resourcesNotToExpectAfterOp: r,
							expectedStatusCode:          http.StatusOK,
						}, fmt.Sprintf("labelQuery=%[1]s eq %[3]s and %[2]s eq %[3]s", commonLabelKey, resourceSpecificLabel, commonLabelValue), ctx.SMWithOAuthForTenant)
					})

					It("returns only tenant specific resources without label query", func() {
						verifyListOpWithAuth(listOpEntry{
							resourcesToExpectBeforeOp: []common.Object{rForTenant},
							resourcesToExpectAfterOp:  []common.Object{rForTenant},
							expectedStatusCode:        http.StatusOK,
						}, "", ctx.SMWithOAuthForTenant)
					})

					Context("when authenticating with global token", func() {
						It("it returns all resources", func() {
							verifyListOpWithAuth(listOpEntry{
								resourcesToExpectBeforeOp: []common.Object{r[0], r[1], rForTenant},
								resourcesToExpectAfterOp:  []common.Object{r[0], r[1], rForTenant},
								expectedStatusCode:        http.StatusOK,
							}, "", ctx.SMWithOAuth)
						})
					})
				})
			}

			if t.SupportsAsyncOperations {
				Context("when attach_last_operations is truthy", func() {
					var resourceWithOneOperation, resourceWithFewOperations common.Object
					var lastOperationForResourceId string
					var resourceWithOneOperationId, resourceWithFewOperationsId string

					createTestOperation := func(resourceID string, opType types.OperationCategory) string {
						id, err := uuid.NewV4()
						Expect(err).ToNot(HaveOccurred())
						labels := make(map[string][]string)
						_, err = ctx.SMRepository.Create(context.TODO(), &types.Operation{
							Base: types.Base{
								ID:        id.String(),
								CreatedAt: time.Now(),
								UpdatedAt: time.Now(),
								Labels:    labels,
								Ready:     true,
							},
							Description:   "test",
							Type:          opType,
							State:         types.SUCCEEDED,
							ResourceID:    resourceID,
							ResourceType:  types.ObjectType(t.API),
							CorrelationID: id.String(),
						})
						Expect(err).ShouldNot(HaveOccurred())
						return id.String()
					}

					BeforeEach(func() {
						resourceWithOneOperation = t.ResourceBlueprint(ctx, ctx.SMWithOAuth, bool(responseMode))
						resourceWithFewOperations = t.ResourceBlueprint(ctx, ctx.SMWithOAuth, bool(responseMode))
						resourceWithOneOperationId = resourceWithOneOperation["id"].(string)
						resourceWithFewOperationsId = resourceWithFewOperations["id"].(string)
						createTestOperation(resourceWithFewOperationsId, types.CREATE)
						lastOperationForResourceId = createTestOperation(resourceWithFewOperationsId, types.DELETE)
					})

					AfterEach(func() {
						err := common.RemoveResourceByCriterion(ctx, query.ByField(query.InOperator, "resource_id", []string{resourceWithOneOperationId, resourceWithOneOperationId}...), types.OperationType)
						Expect(err).ShouldNot(HaveOccurred())
						err = common.RemoveResourceByCriterion(ctx, query.ByField(query.InOperator, "id", []string{resourceWithOneOperationId, resourceWithOneOperationId}...), t.ResourceType)
						if err != nil {
							print(err.Error())
						}
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("retrieves the resources list when each resource contains the last operation it is associated to", func() {
						resp := ctx.SMWithOAuth.GET(t.API).
							WithQuery("attach_last_operations", "true").
							Expect().Status(http.StatusOK).
							JSON()

						for _, resource := range resp.Path("$.items[*]").Array().Iter() {
							itemResourceId := resource.Object().Value("id").String().Raw()
							if itemResourceId == resourceWithOneOperationId || itemResourceId == resourceWithFewOperationsId {
								resource.Object().ContainsKey("last_operation")
								lastOp := resource.Object().Value("last_operation")
								Expect(lastOp.Object().Value("resource_id").String().Raw()).To(Equal(itemResourceId))
								Expect(lastOp.Object().Value("state").String().Raw()).To(Equal("succeeded"))
								Expect(lastOp.Object().Value("resource_type").String().Raw()).ToNot(BeEmpty())
								Expect(lastOp.Object().Value("type").String().Raw()).ToNot(BeEmpty())
								Expect(lastOp.Object().Value("deletion_scheduled").String().Raw()).ToNot(BeEmpty())

								if itemResourceId == resourceWithFewOperationsId {
									Expect(lastOp.Object().Value("state").String().Raw()).To(Equal("succeeded"))
									Expect(lastOp.Object().Value("type").String().Raw()).To(Equal("delete"))
									Expect(lastOp.Object().Value("id").String().Raw()).To(Equal(lastOperationForResourceId))
								}
							}
						}
					})
				})

				Context("when the last operation does not exist", func() {
					var resource common.Object
					BeforeEach(func() {
						resource = t.ResourceBlueprint(ctx, ctx.SMWithOAuth, bool(responseMode))
					})

					AfterEach(func() {
						criteria := query.ByField(query.EqualsOperator, "id", resource["id"].(string))
						err := common.RemoveResourceByCriterion(ctx, criteria, t.ResourceType)
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("should return the resource without it", func() {
						resourceId := resource["id"].(string)
						criteria := query.ByField(query.EqualsOperator, "resource_id", resourceId)
						err := ctx.SMRepository.Delete(context.Background(), types.OperationType, criteria)
						Expect(err).ShouldNot(HaveOccurred())

						resp := ctx.SMWithOAuth.GET(t.API).
							WithQuery("attach_last_operations", "true").
							Expect().Status(http.StatusOK).
							JSON()

						var foundResource bool
						for _, resource := range resp.Path("$.items[*]").Array().Iter() {
							itemResourceId := resource.Object().Value("id").String().Raw()
							if itemResourceId == resourceId {
								foundResource = true
								resource.Object().NotContainsKey("last_operation")
							}
						}

						Expect(foundResource).To(Equal(true))
					})
				})

				Context("when attach_last_operations is falsy", func() {
					It("retrieves the resources without their corresponding last operations", func() {
						resp := ctx.SMWithOAuth.GET(t.API).
							WithQuery("attach_last_operations", "false").
							Expect().Status(http.StatusOK).
							JSON()
						for _, resource := range resp.Path("$.items[*]").Array().Iter() {
							resource.Object().NotContainsKey("last_operation")
						}
					})
				})
			}

			Context("Paging", func() {
				Context("with max items query", func() {
					It("returns smaller pages token and Link header", func() {
						pageSize := 5
						resp := ctx.SMWithOAuth.GET(t.API).WithQuery("max_items", pageSize).Expect().Status(http.StatusOK)

						resp.Header("Link").Contains(fmt.Sprintf("<%s?max_items=%d&token=", t.API, pageSize)).Contains(`>; rel="next"`)
						resp.JSON().Path("$.num_items").Number().Gt(0)
						resp.JSON().Path("$.items[*]").Array().Length().Gt(0).Le(pageSize)
						resp.JSON().Path("$.token").NotNull()
					})
				})

				Context("with max items query and label query", func() {
					const labelKey = "pagingLabel"
					var pageSize int
					var objID string
					BeforeEach(func() {
						objID = r[len(r)-1]["id"].(string)
						pageSize = len(r) / 2
						patchLabels := []*types.LabelChange{
							{
								Operation: types.AddLabelOperation,
								Key:       labelKey,
								Values:    []string{objID},
							},
						}

						By(fmt.Sprintf("Attempting add one additional %s label with value %v to resoucre of type %s with id %s", labelKey, []string{objID}, t.API, objID))
						t.PatchResource(ctx, t.StrictlyTenantScoped, t.API, objID, types.ObjectType(t.API), patchLabels, bool(responseMode))

						object := ctx.SMWithOAuth.ListWithQuery(t.API, fmt.Sprintf("fieldQuery=id eq '%s'", objID)).First().Object()
						object.Path(fmt.Sprintf("$.labels[%s][*]", labelKey)).Array().Contains(objID)
					})

					It("successfully returns the item", func() {
						array := ctx.SMWithOAuth.ListWithQuery(t.API, fmt.Sprintf("max_items=%d&labelQuery=%s eq '%s'", pageSize, labelKey, objID))
						array.Length().Equal(1)
						array.Path(fmt.Sprintf("$[0].labels[%s][*]", labelKey)).Array().Contains(objID)
					})
				})

				Context("with negative max items query", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.GET(t.API).WithQuery("max_items", -1).Expect().Status(http.StatusBadRequest)
					})
				})
				Context("with non numerical max_items query", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.GET(t.API).WithQuery("max_items", "invalid").Expect().Status(http.StatusBadRequest)
					})
				})
				Context("with zero max items query", func() {
					It("returns count of the items only", func() {
						resp := ctx.SMWithOAuth.GET(t.API).WithQuery("max_items", 0).Expect().Status(http.StatusOK).JSON()

						resp.Object().NotContainsKey("items")
						resp.Path("$.num_items").Number().Gt(0)
					})
				})
				When("there are no more pages", func() {
					It("should not return token and Link header", func() {
						resp := ctx.SMWithOAuth.GET(t.API).WithQuery("max_items", 0).Expect().Status(http.StatusOK)

						resp.JSON().Object().NotContainsKey("items")
						pageSize := resp.JSON().Path("$.num_items").Number().Raw()

						resp = ctx.SMWithOAuth.GET(t.API).WithQuery("max_items", pageSize).Expect().Status(http.StatusOK)

						resp.Header("Link").Empty()
						resp.JSON().Object().NotContainsKey("token")
						resp.JSON().Path("$.num_items").Number().Gt(0)
						resp.JSON().Path("$.items[*]").Array().Length().Gt(0).Le(pageSize)
					})
				})
				Context("with invalid token", func() {
					executeWithInvalidToken := func(token string) {
						ctx.SMWithOAuth.GET(t.API).WithQuery("token", token).Expect().Status(http.StatusBadRequest)
					}
					Context("no base64 encoded", func() {
						It("returns 404", func() {
							executeWithInvalidToken("invalid")
						})
					})
					Context("non numerical", func() {
						It("returns 404", func() {
							token := base64.StdEncoding.EncodeToString([]byte("non-numerical"))
							executeWithInvalidToken(token)
						})
					})
					Context("negative value", func() {
						It("returns 404", func() {
							token := base64.StdEncoding.EncodeToString([]byte("-1"))
							executeWithInvalidToken(token)
						})
					})
				})
			})

			Context("with no field query", func() {
				It("it returns all resources", func() {
					verifyListOpWithAuth(listOpEntry{
						resourcesToExpectBeforeOp: []common.Object{r[0], r[1]},
						resourcesToExpectAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "", ctx.SMWithOAuth)
				})
			})

			Context("with empty field query", func() {
				It("returns 200", func() {
					verifyListOp(listOpEntry{
						resourcesToExpectBeforeOp: []common.Object{r[0], r[1]},
						resourcesToExpectAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "fieldQuery=")
				})
			})

			Context("with empty label query", func() {
				It("returns 200", func() {
					verifyListOp(listOpEntry{
						resourcesToExpectBeforeOp: []common.Object{r[0], r[1]},
						resourcesToExpectAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "labelQuery=")
				})
			})

			Context("with empty label query and field query", func() {
				It("returns 200", func() {
					verifyListOp(listOpEntry{
						resourcesToExpectBeforeOp: []common.Object{r[0], r[1]},
						resourcesToExpectAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "labelQuery=&fieldQuery=")
				})
			})

			// expand all field and label query test enties into Its wrapped by descriptive Contexts
			for i := 0; i < len(entries); i++ {
				params := entries[i].Parameters[0].(listOpEntry)
				if len(params.queryTemplate) == 0 {
					panic("query templates missing")
				}
				var multiQueryValue string
				var queryValues []string

				fields := common.CopyObject(params.queryArgs)
				delete(fields, "labels")
				multiQueryValue, queryValues = expandFieldQuery(fields, params.queryTemplate)
				fquery := "fieldQuery" + "=" + multiQueryValue

				Context("with field query=", func() {
					for _, queryValue := range queryValues {
						query := "fieldQuery" + "=" + queryValue
						DescribeTable(queryValue, func(test listOpEntry) {
							verifyListOp(test, query)
						}, entries[i])
					}

					if len(queryValues) > 1 {
						DescribeTable(multiQueryValue, func(test listOpEntry) {
							verifyListOp(test, fquery)
						}, entries[i])

					}
				})

				labels := params.queryArgs["labels"]
				if labels != nil {
					multiQueryValue, queryValues = expandLabelQuery(labels.(map[string]interface{}), params.queryTemplate)
					lquery := "labelQuery" + "=" + multiQueryValue

					Context("with label query=", func() {
						for _, queryValue := range queryValues {
							query := "labelQuery" + "=" + queryValue
							DescribeTable(queryValue, func(test listOpEntry) {
								verifyListOp(test, query)
							}, entries[i])
						}

						if len(queryValues) > 1 {
							DescribeTable(multiQueryValue, func(test listOpEntry) {
								verifyListOp(test, lquery)
							}, entries[i])
						}
					})

					Context("with multiple field and label queries", func() {
						DescribeTable(fquery+"&"+lquery, func(test listOpEntry) {
							verifyListOp(test, fquery+"&"+lquery)
						}, entries[i])
					})
				}
			}
		})
	})
}

func expandFieldQuery(fieldQueryArgs common.Object, queryTemplate string) (string, []string) {
	var expandedMultiQuery string
	var expandedQueries []string
	for k, v := range fieldQueryArgs {
		if v == nil {
			continue
		}

		if m, ok := v.(map[string]interface{}); ok {
			bytes, err := json.Marshal(m)
			if err != nil {
				panic(err)
			}
			v = string(bytes)
		}
		if a, ok := v.([]interface{}); ok {
			bytes, err := json.Marshal(a)
			if err != nil {
				panic(err)
			}
			v = string(bytes)

		}
		expandedQueries = append(expandedQueries, fmt.Sprintf(queryTemplate, k, v))
	}

	expandedMultiQuery = strings.Join(expandedQueries, " and ")
	return expandedMultiQuery, expandedQueries
}

func expandLabelQuery(labelQueryArgs map[string]interface{}, queryTemplate string) (string, []string) {
	var expandedMultiQuery string
	var expandedQueries []string

	for key, values := range labelQueryArgs {
		for _, value := range values.([]interface{}) {
			expandedQueries = append(expandedQueries, fmt.Sprintf(queryTemplate, key, value))
		}
	}

	expandedMultiQuery = strings.Join(expandedQueries, " and ")
	return expandedMultiQuery, expandedQueries
}
