package notification_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/storage/interceptors"

	"github.com/Peripli/service-manager/storage/catalog"

	. "github.com/benjamintf1/unmarshalledmatchers"

	"github.com/tidwall/gjson"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
)

func TestNotifications(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Notifications Suite")
}

type notificationTypeEntry struct {
	// ResourceType is the resource object type
	ResourceType types.ObjectType
	// ResourceCreateFunc is blueprint for resource creation
	ResourceCreateFunc func() common.Object
	// ResourceUpdateFunc is blueprint for resource update
	ResourceUpdateFunc func(obj common.Object, update common.Object) common.Object
	// ResourceUpdates are the test updates to be performed (returns the update body json)
	ResourceUpdates []func() common.Object
	// ResourceDeleteFunc is blueprint for resource deletion
	ResourceDeleteFunc func(obj common.Object)
	// ExpectedPlatformIDFunc calculates the expected platform ID for the given object
	ExpectedPlatformIDFunc func(obj common.Object) string
	// ExpectedAdditionalPayloadFunc calculates the expected additional payload for the given object
	ExpectedAdditionalPayloadFunc func(expected common.Object, repository storage.Repository) []byte
	// Verify additional stuff such as creation of notifications for dependant entities
	AdditionalVerificationNotificationsFunc func(expected common.Object, repository storage.Repository, notificationsAfterOp *types.Notifications)
}

var _ = Describe("Notifications Suite", func() {
	var ctx *common.TestContext
	var c context.Context
	var objAfterOp common.Object

	entries := []notificationTypeEntry{
		{
			ResourceType: types.ServiceBrokerType,
			ResourceCreateFunc: func() common.Object {
				_, obj, _ := ctx.RegisterBroker()
				delete(obj, "credentials")
				return obj
			},
			ResourceUpdateFunc: func(obj common.Object, update common.Object) common.Object {
				patchedObj := ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + obj["id"].(string)).
					WithJSON(update).
					Expect().
					Status(http.StatusOK).JSON().Object().Raw()

				return patchedObj
			},
			ResourceUpdates: []func() common.Object{
				func() common.Object {
					return common.Object{}
				},
				func() common.Object {
					return common.Object{
						"description": "test",
					}
				},
				func() common.Object {
					anotherPlatformID := ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(common.Object{
						"id":          "cluster1",
						"name":        "k8s1",
						"type":        "kubernetes",
						"description": "description1",
					}).Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
					Expect(anotherPlatformID).ToNot(BeEmpty())

					return common.Object{
						"platform_id": anotherPlatformID,
					}
				},
				func() common.Object {
					return common.Object{
						"labels": common.Array{
							common.Object{
								"op":  "add",
								"key": "test",
								"values": common.Array{
									"test",
								},
							},
							common.Object{
								"op":  "remove_value",
								"key": "test",
								"values": common.Array{
									"test",
								},
							},
						},
					}
				},
			},
			ResourceDeleteFunc: func(object common.Object) {
				ctx.CleanupBroker(object["id"].(string))
			},
			ExpectedPlatformIDFunc: func(object common.Object) string {
				return ""
			},
			ExpectedAdditionalPayloadFunc: func(expected common.Object, repository storage.Repository) []byte {
				serviceOfferings, err := catalog.Load(c, expected["id"].(string), ctx.SMRepository)
				Expect(err).ShouldNot(HaveOccurred())

				bytes, err := json.Marshal(struct {
					ServiceOfferings []*types.ServiceOffering `json:"services"`
				}{ServiceOfferings: serviceOfferings.ServiceOfferings})
				Expect(err).ShouldNot(HaveOccurred())

				return bytes
			},
			AdditionalVerificationNotificationsFunc: func(expected common.Object, repository storage.Repository, notificationsAfterOp *types.Notifications) {
				serviceOfferings, err := catalog.Load(c, expected["id"].(string), ctx.SMRepository)
				Expect(err).ShouldNot(HaveOccurred())

				for _, serviceOffering := range serviceOfferings.ServiceOfferings {
					for _, servicePlan := range serviceOffering.Plans {
						if servicePlan.Free {
							found := false
							for _, notification := range notificationsAfterOp.Notifications {
								if notification.Resource == types.VisibilityType && notification.Type == types.CREATED {
									catalogID := gjson.GetBytes(notification.Payload, "new.additional.service_plan.catalog_id").Str
									Expect(catalogID).ToNot(BeEmpty())

									if servicePlan.CatalogID == catalogID {
										found = true
										break
									}
								}
							}
							if !found {
								Fail(fmt.Sprintf("Could not find notification for visibility of public plan with SM ID %s and catalog ID %s", servicePlan.ID, servicePlan.CatalogID))
							}
						}
					}
				}

			},
		},
		{
			ResourceType: types.VisibilityType,
			ResourceCreateFunc: func() common.Object {
				visReqBody := make(common.Object, 0)
				cPaidPlan := common.GeneratePaidTestPlan()
				cService := common.GenerateTestServiceWithPlans(cPaidPlan)
				catalog := common.NewEmptySBCatalog()
				catalog.AddService(cService)
				id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

				object := ctx.SMWithOAuth.GET("/v1/service_offerings").WithQuery("fieldQuery", "broker_id = "+id).
					Expect()

				so := object.Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().First()

				servicePlanID := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("fieldQuery", fmt.Sprintf("service_offering_id = %s", so.Object().Value("id").String().Raw())).
					Expect().
					Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First().Object().Value("id").String().Raw()
				visReqBody["service_plan_id"] = servicePlanID

				platformID := ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(common.GenerateRandomPlatform()).
					Expect().
					Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
				visReqBody["platform_id"] = platformID

				visibility := ctx.SMWithOAuth.POST("/v1/visibilities").WithJSON(visReqBody).Expect().
					Status(http.StatusCreated).JSON().Object().Raw()

				return visibility
			},
			ResourceUpdateFunc: func(obj common.Object, update common.Object) common.Object {
				updatedObj := ctx.SMWithOAuth.PATCH("/v1/visibilities/" + obj["id"].(string)).WithJSON(update).Expect().
					Status(http.StatusOK).JSON().Object().Raw()

				return updatedObj
			},
			ResourceUpdates: []func() common.Object{
				func() common.Object {
					return common.Object{}
				},
				func() common.Object {
					return common.Object{
						"description": "test",
					}
				},
				func() common.Object {
					anotherPlatformID := ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(common.Object{
						"id":          "cluster123",
						"name":        "k8s123s",
						"type":        "kubernetes",
						"description": "description1",
					}).Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
					Expect(anotherPlatformID).ToNot(BeEmpty())

					return common.Object{
						"platform_id": anotherPlatformID,
					}
				},
				func() common.Object {
					return common.Object{
						"labels": common.Array{
							common.Object{
								"op":  "add",
								"key": "test",
								"values": common.Array{
									"test",
								},
							},
							common.Object{
								"op":  "remove_value",
								"key": "test",
								"values": common.Array{
									"test",
								},
							},
						},
					}
				},
			},
			ResourceDeleteFunc: func(obj common.Object) {
				ctx.SMWithOAuth.DELETE("/v1/visibilities/" + obj["id"].(string)).Expect().
					Status(http.StatusOK)
			},
			ExpectedPlatformIDFunc: func(obj common.Object) string {
				return obj["platform_id"].(string)
			},
			ExpectedAdditionalPayloadFunc: func(expected common.Object, repository storage.Repository) []byte {
				expectedPlan, err := repository.Get(c, types.ServicePlanType, expected["service_plan_id"].(string))
				Expect(err).ShouldNot(HaveOccurred())
				expectedServicePlan := expectedPlan.(*types.ServicePlan)

				service, err := repository.Get(c, types.ServiceOfferingType, expectedServicePlan.ServiceOfferingID)
				Expect(err).ShouldNot(HaveOccurred())
				serviceOffering := service.(*types.ServiceOffering)

				broker, err := repository.Get(c, types.ServiceBrokerType, serviceOffering.BrokerID)
				Expect(err).ShouldNot(HaveOccurred())

				serviceBroker := broker.(*types.ServiceBroker)

				bytes, err := json.Marshal(interceptors.VisibilityAdditional{
					BrokerID:    serviceBroker.ID,
					BrokerName:  serviceBroker.Name,
					ServicePlan: expectedServicePlan,
				})
				Expect(err).ShouldNot(HaveOccurred())

				return bytes
			},
			AdditionalVerificationNotificationsFunc: func(expected common.Object, repository storage.Repository, notificationsAfterOp *types.Notifications) {

			},
		},
	}

	BeforeSuite(func() {
		// Register the public plans interceptor with default public plans function that uses the catalog plan free value
		// so that we can verify that notifications for public plans are also created
		ctx = common.NewTestContextBuilder().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			smb.WithCreateInterceptorProvider(types.ServiceBrokerType, &interceptors.PublicPlanCreateInterceptorProvider{
				IsCatalogPlanPublicFunc: func(broker *types.ServiceBroker, catalogService *types.ServiceOffering, catalogPlan *types.ServicePlan) (b bool, e error) {
					return catalogPlan.Free, nil
				},
			}).OnTxBefore(interceptors.BrokerCreateNotificationInterceptorName).Register()

			smb.WithUpdateInterceptorProvider(types.ServiceBrokerType, &interceptors.PublicPlanUpdateInterceptorProvider{
				IsCatalogPlanPublicFunc: func(broker *types.ServiceBroker, catalogService *types.ServiceOffering, catalogPlan *types.ServicePlan) (b bool, e error) {
					return catalogPlan.Free, nil
				},
			}).OnTxBefore(interceptors.BrokerUpdateNotificationInterceptorName).Register()

			return nil
		}).Build()
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	BeforeEach(func() {
		c = context.TODO()
		objAfterOp = nil
	})

	for _, entry := range entries {
		entry := entry

		getNotifications := func(ids ...string) (*types.Notifications, []string) {
			filters := make([]query.Criterion, 0)
			if len(ids) != 0 {
				filters = append(filters, query.ByField(query.NotInOperator, "id", "["+strings.Join(ids, "||"+"]")))
			}

			objectList, err := ctx.SMRepository.List(c, types.NotificationType, filters...)
			Expect(err).ShouldNot(HaveOccurred())

			notifications := objectList.(*types.Notifications)
			notificatonIDs := make([]string, 0, len(notifications.Notifications))
			for _, n := range notifications.Notifications {
				notificatonIDs = append(notificatonIDs, n.GetID())
			}

			return notifications, notificatonIDs
		}

		verifyCreationNotificationCreated := func(objAfterOp common.Object, notificationsAfterOp *types.Notifications) {
			found := false
			for _, notification := range notificationsAfterOp.Notifications {
				if notification.Type != types.CREATED {
					continue
				}

				newObjID := gjson.GetBytes(notification.Payload, "new.resource.id").String()
				if newObjID != objAfterOp["id"] {
					continue
				}

				if notification.PlatformID != entry.ExpectedPlatformIDFunc(objAfterOp) {
					continue
				}

				resource := gjson.GetBytes(notification.Payload, "new.resource").Value().(common.Object)
				Expect(resource).To(Equal(objAfterOp))

				actualPayload := gjson.GetBytes(notification.Payload, "new.additional").Raw
				expectedPayload := entry.ExpectedAdditionalPayloadFunc(objAfterOp, ctx.SMRepository)
				Expect(actualPayload).To(MatchUnorderedJSON(expectedPayload))
				found = true
			}

			if !found {
				Fail(fmt.Sprintf("Expected to find notification for resource type %s", entry.ResourceType))
			}
		}

		verifyDeletionNotificationCreated := func(objAfterOp common.Object, notificationsAfterOp *types.Notifications, expectedOldPayload []byte) {
			found := false
			for _, notification := range notificationsAfterOp.Notifications {
				if notification.Type != types.DELETED {
					continue
				}

				oldObjID := gjson.GetBytes(notification.Payload, "old.resource.id").String()
				if oldObjID != objAfterOp["id"] {
					continue
				}

				if notification.PlatformID != entry.ExpectedPlatformIDFunc(objAfterOp) {
					continue
				}

				resource := gjson.GetBytes(notification.Payload, "old.resource").Value().(common.Object)
				Expect(resource).To(Equal(objAfterOp))

				actualPayload := gjson.GetBytes(notification.Payload, "old.additional").Raw
				Expect(actualPayload).To(MatchUnorderedJSON(expectedOldPayload))

				found = true
			}

			if !found {
				Fail(fmt.Sprintf("Expected to find notification for resource type %s", entry.ResourceType))
			}
		}

		verifyModificationNotificationsCreated := func(objBeforeOp, objAfterOp, update common.Object, notificationsAfterOp *types.Notifications) {
			found := false
			var expectedOldPayload []byte
			for _, notification := range notificationsAfterOp.Notifications {
				if notification.Type != types.MODIFIED {
					continue
				}

				oldObjID := gjson.GetBytes(notification.Payload, "old.resource.id").String()
				newObjID := gjson.GetBytes(notification.Payload, "new.resource.id").String()

				if oldObjID != objBeforeOp["id"] || newObjID != objAfterOp["id"] {
					continue
				}

				if notification.PlatformID != entry.ExpectedPlatformIDFunc(objAfterOp) {
					continue
				}

				oldResource := gjson.GetBytes(notification.Payload, "old.resource").Value().(common.Object)
				Expect(oldResource).To(Equal(objBeforeOp))

				actualOldPayload := gjson.GetBytes(notification.Payload, "old.additional").Raw
				expectedOldPayload = entry.ExpectedAdditionalPayloadFunc(objBeforeOp, ctx.SMRepository)
				Expect(actualOldPayload).To(MatchUnorderedJSON(expectedOldPayload))

				newResource := gjson.GetBytes(notification.Payload, "new.resource").Value().(common.Object)
				Expect(newResource).To(Equal(objAfterOp))

				actualNewPayload := gjson.GetBytes(notification.Payload, "new.additional").Raw
				expectedNewPayload := entry.ExpectedAdditionalPayloadFunc(objAfterOp, ctx.SMRepository)
				Expect(actualNewPayload).To(MatchUnorderedJSON(expectedNewPayload))

				if labels, ok := update["labels"]; ok {
					labelsJSON := gjson.GetBytes(notification.Payload, "label_changes").Raw
					labelBytes, err := json.Marshal(labels)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(labelsJSON).To(MatchUnorderedJSON(labelBytes))
				}

				found = true
			}

			if !found {
				Fail(fmt.Sprintf("Expected to find notification for resource type %s", entry.ResourceType))
			}

			if entry.ExpectedPlatformIDFunc(objBeforeOp) != entry.ExpectedPlatformIDFunc(objAfterOp) {
				verifyCreationNotificationCreated(objAfterOp, notificationsAfterOp)
				verifyDeletionNotificationCreated(objBeforeOp, notificationsAfterOp, expectedOldPayload)
			}
		}

		Context(fmt.Sprintf("ON %s resource CREATE", entry.ResourceType), func() {
			AfterEach(func() {
				entry.ResourceDeleteFunc(objAfterOp)
			})

			It("also creates a CREATED notification", func() {
				_, ids := getNotifications()
				objAfterOp = entry.ResourceCreateFunc()
				notificationsAfterOp, _ := getNotifications(ids...)

				verifyCreationNotificationCreated(objAfterOp, notificationsAfterOp)
				entry.AdditionalVerificationNotificationsFunc(objAfterOp, ctx.SMRepository, notificationsAfterOp)
			})
		})

		Context(fmt.Sprintf("ON %s resource DELETE", entry.ResourceType), func() {
			BeforeEach(func() {
				objAfterOp = entry.ResourceCreateFunc()
			})

			It("also creates a DELETED notification", func() {
				_, ids := getNotifications()
				oldPayload := entry.ExpectedAdditionalPayloadFunc(objAfterOp, ctx.SMRepository)

				entry.ResourceDeleteFunc(objAfterOp)
				notificationsAfterOp, _ := getNotifications(ids...)

				verifyDeletionNotificationCreated(objAfterOp, notificationsAfterOp, oldPayload)
			})
		})

		Context(fmt.Sprintf("ON %s resource UPDATE", entry.ResourceType), func() {
			var createdObj common.Object

			BeforeEach(func() {
				createdObj = entry.ResourceCreateFunc()
			})

			AfterEach(func() {
				entry.ResourceDeleteFunc(createdObj)
			})

			updateOpEntries := func(updates []func() common.Object) []TableEntry {
				entries := make([]TableEntry, 0, len(updates))

				for i, update := range updates {
					entries = append(entries, Entry(fmt.Sprintf("update # %d", i+1), update))
				}

				return entries
			}

			DescribeTable("also creates one or more notifications", func(update func() common.Object) {
				_, ids := getNotifications()
				updateBody := update()
				objAfterOp = entry.ResourceUpdateFunc(createdObj, updateBody)
				notificationsAfterOp, _ := getNotifications(ids...)

				verifyModificationNotificationsCreated(createdObj, objAfterOp, updateBody, notificationsAfterOp)

			}, updateOpEntries(entry.ResourceUpdates)...)
		})
	}
})
