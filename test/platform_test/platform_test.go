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

package platform_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/storage"
	"github.com/gavv/httpexpect"
	"github.com/gofrs/uuid"
	"github.com/tidwall/gjson"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/Peripli/service-manager/pkg/web"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/test"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

// TestPlatforms tests for platform API
func TestPlatforms(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Platform API Tests Suite")
}

var _ = test.DescribeTestsFor(test.TestCase{
	API: web.PlatformsURL,
	SupportedOps: []test.Op{
		test.Get, test.List, test.Delete, test.DeleteList, test.Patch,
	},
	MultitenancySettings: &test.MultitenancySettings{
		ClientID:           "tenancyClient",
		ClientIDTokenClaim: "cid",
		TenantTokenClaim:   "zid",
		LabelKey:           "tenant",
		TokenClaims: map[string]interface{}{
			"cid": "tenancyClient",
			"zid": "tenantID",
		},
	},
	SupportsAsyncOperations:                false,
	SupportsCascadeDeleteOperations:        true,
	ResourceBlueprint:                      blueprint(true),
	SubResourcesBlueprint:                  subResourcesBlueprint(),
	ResourceWithoutNullableFieldsBlueprint: blueprint(false),
	ResourcePropertiesToIgnore:             []string{"last_operation"},
	PatchResource:                          test.APIResourcePatch,
	AdditionalTests: func(ctx *common.TestContext, t *test.TestCase) {
		Context("non-generic tests", func() {
			BeforeEach(func() {
				common.RemoveAllPlatforms(ctx.SMRepository)
			})

			Describe("POST", func() {
				Context("With 2 platforms", func() {
					BeforeEach(func() {
						platformJSON := common.GenerateRandomPlatform()
						platformJSON["name"] = "k"
						common.RegisterPlatformInSM(platformJSON, ctx.SMWithOAuth, nil)

						platformJSON2 := common.GenerateRandomPlatform()
						platformJSON2["name"] = "a"
						common.RegisterPlatformInSM(platformJSON2, ctx.SMWithOAuth, nil)
					})

					It("should return them ordered by name", func() {
						result, err := ctx.SMRepository.List(context.Background(), types.PlatformType, query.OrderResultBy("name", query.AscOrder))
						Expect(err).ShouldNot(HaveOccurred())
						Expect(result.Len()).To(BeNumerically(">=", 2))
						names := make([]string, 0, result.Len())
						for i := 0; i < result.Len(); i++ {
							names = append(names, result.ItemAt(i).(*types.Platform).Name)
						}
						Expect(sort.StringsAreSorted(names)).To(BeTrue())
					})

					It("should limit result to only 1", func() {
						result, err := ctx.SMRepository.List(context.Background(), types.PlatformType, query.LimitResultBy(1))
						Expect(err).ShouldNot(HaveOccurred())
						Expect(result.Len()).To(Equal(1))
					})
				})

				Context("when content type is not JSON", func() {
					It("returns 415", func() {
						ctx.SMWithOAuth.POST(web.PlatformsURL).
							WithText("text").
							Expect().Status(http.StatusUnsupportedMediaType)
					})
				})

				Context("when request body is not a valid JSON", func() {
					It("returns 400 if input is not valid JSON", func() {
						ctx.SMWithOAuth.POST(web.PlatformsURL).
							WithText("invalid json").
							WithHeader("content-type", "application/json").
							Expect().Status(http.StatusBadRequest)
					})
				})

				Context("With missing mandatory fields", func() {
					It("returns 400", func() {
						newplatform := func() common.Object {
							return common.MakePlatform("platform1", "cf-10", "cf", "descr")
						}
						ctx.SMWithOAuth.POST(web.PlatformsURL).
							WithJSON(newplatform()).
							Expect().Status(http.StatusCreated)

						for _, prop := range []string{"name", "type"} {
							platform := newplatform()
							delete(platform, prop)

							ctx.SMWithOAuth.POST(web.PlatformsURL).
								WithJSON(platform).
								Expect().Status(http.StatusBadRequest)
						}
					})
				})

				Context("With conflicting fields", func() {
					It("returns 409", func() {
						platform := common.MakePlatform("platform1", "cf-10", "cf", "descr")
						ctx.SMWithOAuth.POST(web.PlatformsURL).
							WithJSON(platform).
							Expect().Status(http.StatusCreated)
						ctx.SMWithOAuth.POST(web.PlatformsURL).
							WithJSON(platform).
							Expect().Status(http.StatusConflict)
					})
				})

				Context("With optional fields skipped", func() {
					It("succeeds", func() {
						platform := common.MakePlatform("platform1", "cf-10", "cf", "descr")
						// delete optional fields
						delete(platform, "id")
						delete(platform, "description")

						reply := ctx.SMWithOAuth.POST(web.PlatformsURL).
							WithJSON(platform).
							Expect().Status(http.StatusCreated).JSON().Object()

						platform["id"] = reply.Value("id").String().Raw()
						// optional fields returned with default values
						platform["description"] = ""

						common.MapContains(reply.Raw(), platform)
					})
				})

				Context("With invalid id", func() {
					It("fails", func() {
						platform := common.MakePlatform("platform/1", "cf-10", "cf", "descr")

						reply := ctx.SMWithOAuth.POST(web.PlatformsURL).
							WithJSON(platform).
							Expect().Status(http.StatusBadRequest).JSON().Object()

						reply.Value("description").Equal("platform/1 contains invalid character(s)")
					})
				})

				Context("Without id", func() {
					It("returns the new platform with generated id and credentials", func() {
						platform := common.MakePlatform("", "cf-10", "cf", "descr")
						delete(platform, "id")

						By("POST returns the new platform")

						reply := ctx.SMWithOAuth.POST(web.PlatformsURL).
							WithJSON(platform).
							Expect().Status(http.StatusCreated).JSON().Object()

						id := reply.Value("id").String().NotEmpty().Raw()
						platform["id"] = id
						common.MapContains(reply.Raw(), platform)
						basic := reply.Value("credentials").Object().Value("basic").Object()
						basic.Value("username").String().NotEmpty()
						basic.Value("password").String().NotEmpty()

						By("GET returns the same platform")

						reply = ctx.SMWithOAuth.GET(web.PlatformsURL + "/" + id).
							Expect().Status(http.StatusOK).JSON().Object()

						common.MapContains(reply.Raw(), platform)
					})
				})

				Context("With async query param", func() {
					It("fails", func() {
						platform := common.MakePlatform("", "cf-10", "cf", "descr")
						delete(platform, "id")

						reply := ctx.SMWithOAuth.POST(web.PlatformsURL).
							WithQuery("async", "true").
							WithJSON(platform).
							Expect().Status(http.StatusBadRequest).JSON().Object()

						reply.Value("description").String().Contains("api doesn't support asynchronous operations")
					})
				})

				Context("Technical Platform", func() {
					AfterEach(func() {
						ctx.SMWithOAuthForTenant.DELETE(web.PlatformsURL + "/1234").
							Expect().Status(http.StatusOK)
					})

					It("should succeed", func() {
						result := ctx.SMWithOAuthForTenant.POST(web.PlatformsURL).
							WithJSON(common.Object{
								"name":        "technical",
								"technical":   true,
								"type":        "kubernetes",
								"description": "none",
								"id":          "1234",
							}).Expect().Status(http.StatusCreated).JSON().Object()
						result.Value("id").String().Equal("1234")
						result.NotContainsKey("credentials")
					})
				})
			})

			Describe("PATCH", func() {
				var platform common.Object
				var platformUser string
				var platformPassword string
				const id = "p1"

				BeforeEach(func() {
					By("Create new platform")

					platform = common.MakePlatform(id, "cf-10", "cf", "descr")
					reply := ctx.SMWithOAuth.POST(web.PlatformsURL).
						WithJSON(platform).
						Expect().Status(http.StatusCreated).JSON().Object()
					basic := reply.Value("credentials").Object().Value("basic").Object()
					platformUser = basic.Value("username").String().Raw()
					platformPassword = basic.Value("password").String().Raw()
				})

				Context("With all properties updated", func() {
					It("returns 200", func() {
						By("Update platform")

						updatedPlatform := common.MakePlatform("", "cf-11", "cff", "descr2")
						delete(updatedPlatform, "id")

						reply := ctx.SMWithOAuth.PATCH(web.PlatformsURL + "/" + id).
							WithJSON(updatedPlatform).
							Expect().
							Status(http.StatusOK).JSON().Object()

						reply.NotContainsKey("credentials")

						updatedPlatform["id"] = id
						common.MapContains(reply.Raw(), updatedPlatform)

						By("Update is persisted")

						reply = ctx.SMWithOAuth.GET(web.PlatformsURL + "/" + id).
							Expect().
							Status(http.StatusOK).JSON().Object()

						common.MapContains(reply.Raw(), updatedPlatform)
					})
				})

				Context("With created_at in body", func() {
					It("should not update created_at", func() {
						By("Update platform")

						createdAt := "2015-01-01T00:00:00Z"
						updatedPlatform := common.Object{
							"created_at": createdAt,
						}

						ctx.SMWithOAuth.PATCH(web.PlatformsURL+"/"+id).
							WithJSON(updatedPlatform).
							Expect().
							Status(http.StatusOK).JSON().Object().
							ContainsKey("created_at").
							NotContainsKey("credentials").
							ValueNotEqual("created_at", createdAt)

						By("Update is persisted")

						ctx.SMWithOAuth.GET(web.PlatformsURL+"/"+id).
							Expect().
							Status(http.StatusOK).JSON().Object().
							ContainsKey("created_at").
							ValueNotEqual("created_at", createdAt)
					})
				})

				Context("With properties updated separately", func() {
					It("returns 200", func() {
						updatedPlatform := common.MakePlatform("", "cf-11", "cff", "descr2")
						delete(updatedPlatform, "id")

						for prop, val := range updatedPlatform {
							update := common.Object{}
							update[prop] = val
							reply := ctx.SMWithOAuth.PATCH(web.PlatformsURL + "/" + id).
								WithJSON(update).
								Expect().
								Status(http.StatusOK).JSON().Object()

							reply.NotContainsKey("credentials")

							platform[prop] = val
							common.MapContains(reply.Raw(), platform)

							reply = ctx.SMWithOAuth.GET(web.PlatformsURL + "/" + id).
								Expect().
								Status(http.StatusOK).JSON().Object()

							common.MapContains(reply.Raw(), platform)
						}
					})
				})

				Context("With provided id", func() {
					It("should not update platform id", func() {
						ctx.SMWithOAuth.PATCH(web.PlatformsURL + "/" + id).
							WithJSON(common.Object{"id": "123"}).
							Expect().
							Status(http.StatusOK).JSON().Object().
							NotContainsKey("credentials")

						ctx.SMWithOAuth.GET(web.PlatformsURL + "/123").
							Expect().
							Status(http.StatusNotFound)
					})
				})

				Context("On missing platform", func() {
					It("returns 404", func() {
						ctx.SMWithOAuth.PATCH(web.PlatformsURL + "/123").
							WithJSON(common.Object{"name": "123"}).
							Expect().
							Status(http.StatusNotFound)
					})
				})

				Context("With conflicting fields", func() {
					It("should return 409", func() {
						platform2 := common.MakePlatform("p2", "cf-12", "cf2", "descr2")
						ctx.SMWithOAuth.POST(web.PlatformsURL).
							WithJSON(platform2).
							Expect().Status(http.StatusCreated)

						ctx.SMWithOAuth.PATCH(web.PlatformsURL + "/" + id).
							WithJSON(platform2).
							Expect().
							Status(http.StatusConflict)
					})
				})

				Context("With regenerate credentials query param", func() {
					getPlatformFromDB := func() *types.Platform {
						platformObj, err := ctx.SMRepository.Get(context.Background(), types.PlatformType, query.ByField(query.EqualsOperator, "id", id))
						Expect(err).NotTo(HaveOccurred())
						dbPlatform := platformObj.(*types.Platform)
						//unmarshaling from json in order to validate this functionality
						data, err := json.Marshal(dbPlatform)
						Expect(err).ToNot(HaveOccurred())
						pl := &types.Platform{}
						err = json.Unmarshal(data, pl)
						Expect(err).ToNot(HaveOccurred())
						return pl
					}

					activatePlatformCredentials := func(user string, password string) {
						platform := &types.Platform{
							Credentials: &types.Credentials{
								Basic: &types.Basic{
									Username: user,
									Password: password,
								},
							},
						}
						_, _, err := ctx.ConnectWebSocket(platform, nil)
						Expect(err).To(Not(HaveOccurred()))
					}

					tryCredentials := func(user string, password string, status int) {
						basicAuth := &common.SMExpect{Expect: ctx.SM.Builder(func(req *httpexpect.Request) {
							req.WithBasicAuth(user, password).WithClient(ctx.HttpClient)
						})}
						basicAuth.GET(web.PlatformsURL + "/" + id).Expect().Status(status)
					}

					When("credentials generated with additional properties", func() {
						It("should succeed", func() {
							updatedPlatform := common.MakePlatform("", "bla", "cf", "descr2")
							delete(updatedPlatform, "id")

							ctx.SMWithOAuth.PATCH(web.PlatformsURL+"/"+id).
								WithJSON(updatedPlatform).
								WithQuery(filters.RegenerateCredentialsQueryParam, "true").
								Expect().
								Status(http.StatusOK)

							dbPlatform := getPlatformFromDB()
							Expect(dbPlatform.Credentials.Basic.Username).NotTo(Equal(platformUser))
							Expect(dbPlatform.Credentials.Basic.Password).NotTo(Equal(platformPassword))
							Expect(dbPlatform.Name).To(Equal("bla"))
						})
					})

					When("credentials are inactive", func() {
						Context("No old credentials", func() {
							It("should return new credentials", func() {
								ctx.SMWithOAuth.PATCH(web.PlatformsURL+"/"+id).
									WithJSON(common.Object{}).
									WithQuery(filters.RegenerateCredentialsQueryParam, "true").
									Expect().
									Status(http.StatusOK)

								dbPlatform := getPlatformFromDB()
								Expect(dbPlatform.OldCredentials).To(BeNil())
								Expect(dbPlatform.Credentials.Basic.Username).NotTo(Equal(platformUser))
								Expect(dbPlatform.Credentials.Basic.Password).NotTo(Equal(platformPassword))
								tryCredentials(dbPlatform.Credentials.Basic.Username, dbPlatform.Credentials.Basic.Password, http.StatusOK)
							})
						})

						Context("old credentials exist", func() {
							BeforeEach(func() {
								By("generate new credentials and keep the old")
								activatePlatformCredentials(platformUser, platformPassword)
								ctx.SMWithOAuth.PATCH(web.PlatformsURL+"/"+id).
									WithJSON(common.Object{}).
									WithQuery(filters.RegenerateCredentialsQueryParam, "true").
									Expect().
									Status(http.StatusOK)
								dbPlatform := getPlatformFromDB()
								Expect(platformUser).To(Equal(dbPlatform.OldCredentials.Basic.Username))
								Expect(platformPassword).To(Equal(dbPlatform.OldCredentials.Basic.Password))
								Expect(dbPlatform.CredentialsActive).To(BeFalse())
								tryCredentials(dbPlatform.Credentials.Basic.Username, dbPlatform.Credentials.Basic.Password, http.StatusOK)
								tryCredentials(dbPlatform.OldCredentials.Basic.Username, dbPlatform.OldCredentials.Basic.Password, http.StatusOK)
							})

							It("should not override old credentials", func() {
								ctx.SMWithOAuth.PATCH(web.PlatformsURL+"/"+id).
									WithJSON(common.Object{}).
									WithQuery(filters.RegenerateCredentialsQueryParam, "true").
									Expect().
									Status(http.StatusOK)

								dbPlatform := getPlatformFromDB()
								Expect(platformUser).To(Equal(dbPlatform.OldCredentials.Basic.Username))
								Expect(platformPassword).To(Equal(dbPlatform.OldCredentials.Basic.Password))
								tryCredentials(dbPlatform.Credentials.Basic.Username, dbPlatform.Credentials.Basic.Password, http.StatusOK)
								tryCredentials(dbPlatform.OldCredentials.Basic.Username, dbPlatform.OldCredentials.Basic.Password, http.StatusOK)
							})
						})
					})

					When("credentials are active", func() {
						BeforeEach(func() {
							activatePlatformCredentials(platformUser, platformPassword)
							dbPlatform := getPlatformFromDB()
							Expect(dbPlatform.OldCredentials).To(BeNil())
							Expect(dbPlatform.CredentialsActive).To(BeTrue())
						})

						It("should sanitize sensitive from response", func() {
							reply := ctx.SMWithOAuth.GET(web.PlatformsURL + "/" + id).
								Expect().
								Status(http.StatusOK).JSON().Object()
							reply.NotContainsKey("credentials")
							reply.NotContainsKey("credentials_active")
						})

						It("should return new credentials and keep current as old", func() {
							ctx.SMWithOAuth.PATCH(web.PlatformsURL+"/"+id).
								WithJSON(common.Object{}).
								WithQuery(filters.RegenerateCredentialsQueryParam, "true").
								Expect().
								Status(http.StatusOK)
							dbPlatform := getPlatformFromDB()
							Expect(dbPlatform.Credentials.Basic.Username).NotTo(Equal(platformUser))
							Expect(dbPlatform.Credentials.Basic.Password).NotTo(Equal(platformPassword))
							Expect(dbPlatform.OldCredentials.Basic.Username).To(Equal(platformUser))
							Expect(dbPlatform.OldCredentials.Basic.Password).To(Equal(platformPassword))
							Expect(dbPlatform.CredentialsActive).To(BeFalse())
							tryCredentials(dbPlatform.Credentials.Basic.Username, dbPlatform.Credentials.Basic.Password, http.StatusOK)
							tryCredentials(dbPlatform.OldCredentials.Basic.Username, dbPlatform.OldCredentials.Basic.Password, http.StatusOK)
						})

						It("old credentials are not usable", func() {
							By("move new credentials to be old")
							reply := ctx.SMWithOAuth.PATCH(web.PlatformsURL+"/"+id).
								WithJSON(common.Object{}).
								WithQuery(filters.RegenerateCredentialsQueryParam, "true").
								Expect().
								Status(http.StatusOK).JSON().Object()

							By("activate and remove old credentials")
							basic := reply.Value("credentials").Object().Value("basic").Object()
							newUser := basic.Value("username").String().Raw()
							newPassword := basic.Value("password").String().Raw()
							activatePlatformCredentials(newUser, newPassword)
							tryCredentials(newUser, newPassword, http.StatusOK)

							By("validate old unusable")
							tryCredentials(platformUser, platformPassword, http.StatusUnauthorized)
						})
					})
				})
			})

			Describe("DELETE", func() {
				const platformID = "p1"
				var platform common.Object

				BeforeEach(func() {
					platform = common.MakePlatform(platformID, "cf-10", "cf", "descr")
					ctx.SMWithOAuth.POST(web.PlatformsURL).
						WithJSON(platform).
						Expect().Status(http.StatusCreated)

					common.CreateInstanceInPlatform(ctx, platformID)
				})

				AfterEach(func() {
					ctx.CleanupAdditionalResources()
				})

				Context("with existing service instances", func() {
					It("should return 400 with user-friendly message", func() {
						ctx.SMWithOAuth.DELETE(web.PlatformsURL + "/" + platformID).
							Expect().
							Status(http.StatusConflict).
							JSON().Object().
							Value("error").String().Contains("ExistingReferenceEntity")
					})
					It("should delete instances when cascade requested", func() {
						ctx.SMWithOAuth.DELETE(web.PlatformsURL+"/"+platformID).
							WithQuery("cascade", "true").
							Expect().
							Status(http.StatusAccepted)
					})
				})

				Context("with active platform", func() {
					var makePlatformActive = func(platformID string) {
						err := ctx.SMRepository.InTransaction(context.TODO(), func(ctx context.Context, storage storage.Repository) error {
							var updatedPlatform types.Object
							byID := query.ByField(query.EqualsOperator, "id", platformID)
							platformFromStorage, err := storage.Get(ctx, types.PlatformType, byID)
							Expect(err).ToNot(HaveOccurred())

							platformFromStorage.(*types.Platform).Active = true
							if updatedPlatform, err = storage.Update(ctx, platformFromStorage, types.LabelChanges{}); err != nil {
								return err
							}
							Expect(updatedPlatform.(*types.Platform).Active).To(Equal(true))
							return nil
						})
						Expect(err).ToNot(HaveOccurred())
					}
					When("sub-resources are exists", func() {
						JustBeforeEach(func() {
							makePlatformActive(platformID)
						})
						It("should return 422 unprocessable entity for cascade delete", func() {
							ctx.SMWithOAuth.DELETE(web.PlatformsURL+"/"+platformID).
								WithQuery("cascade", "true").
								Expect().
								Status(http.StatusUnprocessableEntity)
						})
					})

					When("Platform active without sub-resources", func() {
						var testPlatformID string
						JustBeforeEach(func() {
							testPlatformID := "platform-with-no-resources"
							ctx.SMWithOAuth.POST(web.PlatformsURL).
								WithJSON(common.MakePlatform(testPlatformID, "platform-with-no-resources", "cf", "descr")).
								Expect().Status(http.StatusCreated)
							makePlatformActive(testPlatformID)
						})
						It("should return ok", func() {
							ctx.SMWithOAuth.DELETE(web.PlatformsURL + "/" + testPlatformID).
								Expect().
								Status(http.StatusOK)
						})
					})

				})

			})

			Describe("GET", func() {
				Context("Technical Platform", func() {
					var opID string
					BeforeEach(func() {
						result := ctx.SMWithOAuthForTenant.POST(web.PlatformsURL).
							WithJSON(common.Object{
								"name":        "technical",
								"technical":   true,
								"type":        "kubernetes",
								"description": "none",
								"id":          "1234",
							}).Expect().Status(http.StatusCreated).JSON().Object()
						opID = result.Value("last_operation").Object().Value("id").String().Raw()
					})

					AfterEach(func() {
						ctx.SMWithOAuthForTenant.DELETE(web.PlatformsURL + "/1234").
							Expect().Status(http.StatusOK)
					})

					It("should be not found", func() {
						ctx.SMWithOAuthForTenant.GET(web.PlatformsURL + "/1234").
							Expect().Status(http.StatusNotFound)
					})

					It("should be filtered out from list", func() {
						result := ctx.SMWithOAuthForTenant.GET(web.PlatformsURL).
							Expect().Status(http.StatusOK).JSON().Object()
						result.NotEmpty().Value("items").Path("$[*].id").Array().
							NotContains("1234")
					})

					It("should not find last operation", func() {
						ctx.SMWithOAuthForTenant.GET(web.PlatformsURL + "/1234/operations/" + opID).
							Expect().Status(http.StatusNotFound)
					})
				})
			})
		})
	},
})

func hashPassword(password string) string {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	return string(passwordHash)
}

func subResourcesBlueprint() func(ctx *common.TestContext, auth *common.SMExpect, async bool, platformID string, resourceType types.ObjectType, platform common.Object) {
	return func(ctx *common.TestContext, auth *common.SMExpect, async bool, platformID string, resourceType types.ObjectType, platform common.Object) {
		var planID, serviceID, brokerID string
		var origBrokerExpect *httpexpect.Expect

		plan := common.GenerateFreeTestPlan()
		planID = gjson.Get(plan, "id").String()

		service := common.GenerateTestServiceWithPlans(plan)
		serviceID = gjson.Get(service, "id").String()

		catalog := common.NewEmptySBCatalog()
		catalog.AddService(service)

		brokerUtils := ctx.RegisterBrokerWithCatalog(catalog)
		brokerID = brokerUtils.Broker.ID

		SMPlatformExpect := ctx.SM.Builder(func(req *httpexpect.Request) {
			username := ctx.TestPlatform.Credentials.Basic.Username
			password := ctx.TestPlatform.Credentials.Basic.Password
			req.WithBasicAuth(username, password)
		})

		SMPlatformExpect.PUT(web.BrokerPlatformCredentialsURL).
			WithJSON(common.Object{
				"broker_id":     brokerID,
				"username":      "admin",
				"password_hash": hashPassword("admin"),
			}).Expect().Status(http.StatusOK)

		common.CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, brokerID)

		origBrokerExpect = ctx.SM.Builder(func(req *httpexpect.Request) {
			req.WithBasicAuth("admin", "admin")
		})

		instanceID1, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}

		instanceID2, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}

		origBrokerExpect.PUT(fmt.Sprintf("%s/%s/v2/service_instances/%s", web.OSBURL, brokerID, instanceID1)).
			WithJSON(common.Object{
				"service_id": serviceID,
				"plan_id":    planID,
				"context": common.Object{
					"platform": "kubernetes",
				},
			}).Expect().Status(http.StatusCreated)

		origBrokerExpect.PUT(fmt.Sprintf("%s/%s/v2/service_instances/%s", web.OSBURL, brokerID, instanceID2)).
			WithJSON(common.Object{
				"service_id": serviceID,
				"plan_id":    planID,
				"context": common.Object{
					"platform": "cloudfoundry",
				},
			}).Expect().Status(http.StatusCreated)

		bindingID1, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}

		bindingID2, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}

		origBrokerExpect.PUT(fmt.Sprintf("%s/%s/v2/service_instances/%s/service_bindings/%s", web.OSBURL, brokerID, instanceID1, bindingID1)).
			WithJSON(common.Object{
				"context":          common.Object{},
				"maintenance_info": common.Object{"version": "old"},
				"parameters":       common.Object{},
				"plan_id":          planID,
				"service_id":       serviceID,
			}).
			Expect().
			Status(http.StatusCreated)

		ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", web.ServiceBindingsURL, bindingID1)).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().
			ContainsMap(map[string]interface{}{
				"id":                  bindingID1,
				"service_instance_id": instanceID1,
			})

		origBrokerExpect.PUT(fmt.Sprintf("%s/%s/v2/service_instances/%s/service_bindings/%s", web.OSBURL, brokerID, instanceID1, bindingID2)).
			WithJSON(common.Object{
				"context":          common.Object{},
				"maintenance_info": common.Object{"version": "old"},
				"parameters":       common.Object{},
				"plan_id":          planID,
				"service_id":       serviceID,
			}).
			Expect().
			Status(http.StatusCreated)

		ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", web.ServiceBindingsURL, bindingID2)).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().
			ContainsMap(map[string]interface{}{
				"id":                  bindingID2,
				"service_instance_id": instanceID1,
			})

	}
}

func blueprint(setNullFieldsValues bool) func(ctx *common.TestContext, auth *common.SMExpect, async bool) common.Object {
	return func(ctx *common.TestContext, auth *common.SMExpect, _ bool) common.Object {
		randomPlatform := common.GenerateRandomPlatform()
		if !setNullFieldsValues {
			delete(randomPlatform, "description")
		}
		reply := auth.POST(web.PlatformsURL).WithJSON(randomPlatform).
			Expect().
			Status(http.StatusCreated).JSON().Object().Raw()
		createdAtString := reply["created_at"].(string)
		updatedAtString := reply["updated_at"].(string)
		createdAt, err := time.Parse(time.RFC3339Nano, createdAtString)
		if err != nil {
			panic(err)
		}
		updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtString)
		if err != nil {
			panic(err)
		}
		platform := &types.Platform{
			Base: types.Base{
				ID:        reply["id"].(string),
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				Ready:     true,
			},
			Credentials: &types.Credentials{
				Basic: &types.Basic{
					Username: reply["credentials"].(map[string]interface{})["basic"].(map[string]interface{})["username"].(string),
					Password: reply["credentials"].(map[string]interface{})["basic"].(map[string]interface{})["password"].(string),
				},
			},
			Type:        reply["type"].(string),
			Description: reply["description"].(string),
			Name:        reply["name"].(string),
		}
		ctx.TestPlatform = platform
		delete(reply, "credentials")
		return reply
	}
}
