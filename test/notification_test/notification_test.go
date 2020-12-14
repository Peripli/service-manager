package notification_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/storage/service_plans"
	"github.com/Peripli/service-manager/test"
	"net/http"
	"testing"
	"time"

	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/tidwall/sjson"

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

const tenantLabelKey = "tenant"

func TestNotifications(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Notifications Suite")
}

type notificationTypeEntry struct {
	// ResourceType is the resource object type
	ResourceType types.ObjectType
	// ResourceTenantScoped whether the resource should be created as tenant-scoped
	ResourceTenantScoped bool
	// ResourceNotifyInactivePlatforms whether notifications for the resource are created for inactive platforms
	ResourceNotifyInactivePlatforms bool
	// ResourceCreateFunc is blueprint for resource creation
	ResourceCreateFunc func() common.Object
	// ResourceUpdateFunc is blueprint for resource update
	ResourceUpdateFunc func(obj common.Object, update common.Object) common.Object
	// ResourceUpdates are the test updates to be performed (returns the update body json)
	ResourceUpdates []func() common.Object
	// ResourceDeleteFunc is blueprint for resource deletion
	ResourceDeleteFunc func(obj common.Object)
	// ExpectedPlatformIDsFunc calculates the expected platform IDs for the given object
	ExpectedPlatformIDsFunc func(obj common.Object) []string
	// ExpectedAdditionalPayloadFunc calculates the expected additional payload for the given object
	ExpectedAdditionalPayloadFunc func(expected common.Object, repository storage.Repository) string
	// Verify additional stuff such as creation of notifications for dependant entities
	AdditionalVerificationNotificationsFunc func(expected common.Object, repository storage.Repository, notificationsAfterOp *types.Notifications)
	ProcessNotificationPayload              func(payload string) string
}

var _ = Describe("Notifications Suite", func() {
	var ctx *common.TestContext
	var c context.Context
	var objAfterOp common.Object
	var otherTenantPlatform *types.Platform

	processBrokersPayload := func(payload string) string {
		var err error
		services := gjson.Get(payload, "services").Raw
		parsed := gjson.Parse(services)
		for i := range parsed.Array() {
			services, err = sjson.Delete(services, fmt.Sprintf("%d.updated_at", i))
			services, err = sjson.Delete(services, fmt.Sprintf("%d.created_at", i))
			services, err = sjson.Delete(services, fmt.Sprintf("%d.ready", i))
			Expect(err).ToNot(HaveOccurred())
			service := gjson.Get(services, fmt.Sprintf("%d", i)).Raw
			plans := gjson.Get(service, "plans")
			for j := range plans.Array() {
				services, err = sjson.Delete(services, fmt.Sprintf("%d.plans.%d.updated_at", i, j))
				services, err = sjson.Delete(services, fmt.Sprintf("%d.plans.%d.created_at", i, j))
				services, err = sjson.Delete(services, fmt.Sprintf("%d.plans.%d.ready", i, j))
				Expect(err).ToNot(HaveOccurred())
			}
		}
		return services
	}

	brokersNotificationEntry := func(tenantScoped bool) notificationTypeEntry {
		var smAuth = func(tenantScoped bool) *common.SMExpect {
			var smAuth *common.SMExpect
			if tenantScoped {
				smAuth = ctx.SMWithOAuthForTenant
			} else {
				smAuth = ctx.SMWithOAuth
			}
			return smAuth
		}

		return notificationTypeEntry{
			ResourceType:         types.ServiceBrokerType,
			ResourceTenantScoped: tenantScoped,
			ResourceCreateFunc: func() common.Object {

				obj := ctx.RegisterBrokerWithCatalogAndLabelsExpect(common.NewRandomSBCatalog(), common.Object{}, smAuth(tenantScoped)).Broker.JSON
				delete(obj, "credentials")
				delete(obj, "last_operation")
				return obj
			},
			ResourceUpdateFunc: func(obj common.Object, update common.Object) common.Object {
				patchedObj := smAuth(tenantScoped).PATCH(web.ServiceBrokersURL + "/" + obj["id"].(string)).
					WithJSON(update).
					Expect().
					Status(http.StatusOK).JSON().Object().Raw()

				delete(patchedObj, "last_operation")
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
					anotherPlatformID := smAuth(tenantScoped).POST(web.PlatformsURL).WithJSON(common.Object{
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
			ExpectedPlatformIDsFunc: func(object common.Object) []string {
				objList, err := ctx.SMRepository.List(context.TODO(), types.PlatformType, query.ByField(query.NotEqualsOperator, "id", types.SMPlatform))
				Expect(err).ToNot(HaveOccurred())

				var brokerTenantID string

				if object["labels"] != nil {
					brokerTenantLabel, found := object["labels"].(common.Object)[tenantLabelKey]
					if found {
						// tenant scoped broker
						brokerTenantID = brokerTenantLabel.(common.Array)[0].(string)
					}
				}

				platformIDs := make([]string, 0)
				for i := 0; i < objList.Len(); i++ {
					platform := objList.ItemAt(i)
					if platformTenantLabel, found := objList.ItemAt(i).GetLabels()[tenantLabelKey]; found && brokerTenantID != "" {
						if len(platformTenantLabel) > 0 && brokerTenantID == platformTenantLabel[0] {
							platformIDs = append(platformIDs, platform.GetID())
						}
					} else {
						platformIDs = append(platformIDs, platform.GetID())
					}
				}

				return platformIDs
			},
			ExpectedAdditionalPayloadFunc: func(expected common.Object, repository storage.Repository) string {
				serviceOfferings, err := catalog.Load(c, expected["id"].(string), ctx.SMRepository)
				Expect(err).ShouldNot(HaveOccurred())

				bytes, err := json.Marshal(struct {
					ServiceOfferings []*types.ServiceOffering `json:"services"`
				}{ServiceOfferings: serviceOfferings.ServiceOfferings})
				Expect(err).ShouldNot(HaveOccurred())

				return processBrokersPayload(string(bytes))
			},
			ProcessNotificationPayload: func(payload string) string {
				return string(processBrokersPayload(payload))
			},
			AdditionalVerificationNotificationsFunc: func(expected common.Object, repository storage.Repository, notificationsAfterOp *types.Notifications) {
				serviceOfferings, err := catalog.Load(c, expected["id"].(string), ctx.SMRepository)
				Expect(err).ShouldNot(HaveOccurred())

				// visibility notifications
				for _, serviceOffering := range serviceOfferings.ServiceOfferings {
					for _, servicePlan := range serviceOffering.Plans {
						if *servicePlan.Free {
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

				// broker notifications
				if tenantScoped {
					// broker notifications
					for _, notification := range notificationsAfterOp.Notifications {
						if notification.Resource == types.ServiceBrokerType {

							platformID := notification.PlatformID
							Expect(platformID).ToNot(BeEmpty())

							if platformID == otherTenantPlatform.ID {
								Fail("Notification of tenant scoped broker was created for another tenant's platform")
							}
						}
					}
				}
			},
		}
	}

	entries := []notificationTypeEntry{
		brokersNotificationEntry(true),
		brokersNotificationEntry(false),
		{
			ResourceType:                    types.VisibilityType,
			ResourceTenantScoped:            false,
			ResourceNotifyInactivePlatforms: true,
			ResourceCreateFunc: func() common.Object {
				visReqBody := make(common.Object)
				cPaidPlan := common.GeneratePaidTestPlan()
				cService := common.GenerateTestServiceWithPlans(cPaidPlan)
				catalog := common.NewEmptySBCatalog()
				catalog.AddService(cService)
				id := ctx.RegisterBrokerWithCatalog(catalog).Broker.ID

				so := ctx.SMWithOAuth.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s'", id)).First()

				servicePlanID := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s'", so.Object().Value("id").String().Raw())).
					First().Object().Value("id").String().Raw()

				labels := types.Labels{
					"organization_guid": []string{"1", "2"},
				}

				visReqBody["service_plan_id"] = servicePlanID
				visReqBody["labels"] = labels

				platformID := ctx.SMWithOAuth.POST(web.PlatformsURL).WithJSON(common.GenerateRandomPlatform()).
					Expect().
					Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
				visReqBody["platform_id"] = platformID

				visibility := ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(visReqBody).Expect().
					Status(http.StatusCreated).JSON().Object().Raw()

				delete(visibility, "last_operation")
				return visibility
			},
			ResourceUpdateFunc: func(obj common.Object, update common.Object) common.Object {
				updatedObj := ctx.SMWithOAuth.PATCH(web.VisibilitiesURL + "/" + obj["id"].(string)).WithJSON(update).Expect().
					Status(http.StatusOK).JSON().Object().Raw()

				delete(updatedObj, "last_operation")
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
					anotherPlatformID := ctx.SMWithOAuth.POST(web.PlatformsURL).WithJSON(common.Object{
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
				func() common.Object {
					return common.Object{
						"labels": common.Array{
							common.Object{
								"op":  "add",
								"key": "organization_guid",
								"values": common.Array{
									"test",
								},
							},
						},
					}
				},
			},
			ResourceDeleteFunc: func(obj common.Object) {
				ctx.SMWithOAuth.DELETE(web.VisibilitiesURL + "/" + obj["id"].(string)).Expect().
					Status(http.StatusOK)
			},
			ExpectedPlatformIDsFunc: func(obj common.Object) []string {
				return []string{obj["platform_id"].(string)}
			},
			ExpectedAdditionalPayloadFunc: func(expected common.Object, repository storage.Repository) string {
				byPlanID := query.ByField(query.EqualsOperator, "id", expected["service_plan_id"].(string))
				expectedPlan, err := repository.Get(c, types.ServicePlanType, byPlanID)
				Expect(err).ShouldNot(HaveOccurred())
				expectedServicePlan := expectedPlan.(*types.ServicePlan)

				byServiceID := query.ByField(query.EqualsOperator, "id", expectedServicePlan.ServiceOfferingID)
				service, err := repository.Get(c, types.ServiceOfferingType, byServiceID)
				Expect(err).ShouldNot(HaveOccurred())
				serviceOffering := service.(*types.ServiceOffering)

				byBrokerID := query.ByField(query.EqualsOperator, "id", serviceOffering.BrokerID)
				broker, err := repository.Get(c, types.ServiceBrokerType, byBrokerID)
				Expect(err).ShouldNot(HaveOccurred())

				serviceBroker := broker.(*types.ServiceBroker)

				bytes, err := json.Marshal(interceptors.VisibilityAdditional{
					BrokerID:    serviceBroker.ID,
					BrokerName:  serviceBroker.Name,
					ServicePlan: expectedServicePlan,
				})
				Expect(err).ShouldNot(HaveOccurred())

				return string(bytes)
			},
			ProcessNotificationPayload: func(payload string) string {
				return payload
			},
			AdditionalVerificationNotificationsFunc: func(expected common.Object, repository storage.Repository, notificationsAfterOp *types.Notifications) {

			},
		},
	}

	multitenancySettings := &test.MultitenancySettings{
		ClientID:           "tenancyClient",
		ClientIDTokenClaim: "cid",
		TenantTokenClaim:   "zid",
		LabelKey:           tenantLabelKey,
		TokenClaims: map[string]interface{}{
			"cid": "tenancyClient",
			"zid": "tenantID",
		},
	}

	BeforeSuite(func() {
		// Register the public plans interceptor with default public plans function that uses the catalog plan free value
		// so that we can verify that notifications for public plans are also created
		ctx = common.NewTestContextBuilderWithSecurity().
			WithTenantTokenClaims(multitenancySettings.TokenClaims).
			WithEnvPostExtensions(func(e env.Environment, servers map[string]common.FakeServer) {
				e.Set("api.protected_labels", multitenancySettings.LabelKey)
			}).
			WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
				_, err := smb.EnableMultitenancy(multitenancySettings.LabelKey, common.ExtractTenantFunc)
				return err
			}).
			WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
				smb.WithCreateInterceptorProvider(types.ServiceBrokerType, &interceptors.PublicPlanCreateInterceptorProvider{
					IsCatalogPlanPublicFunc: func(broker *types.ServiceBroker, catalogService *types.ServiceOffering, catalogPlan *types.ServicePlan) (b bool, e error) {
						return *catalogPlan.Free, nil
					},
					SupportedPlatformsFunc: func(ctx context.Context, plan *types.ServicePlan, repository storage.Repository) (map[string]*types.Platform, error) {
						return service_plans.ResolveSupportedPlatformsForPlans(ctx, []*types.ServicePlan{plan}, repository)
					},
				}).OnTxBefore(interceptors.BrokerCreateNotificationInterceptorName).Register()

				smb.WithUpdateInterceptorProvider(types.ServiceBrokerType, &interceptors.PublicPlanUpdateInterceptorProvider{
					IsCatalogPlanPublicFunc: func(broker *types.ServiceBroker, catalogService *types.ServiceOffering, catalogPlan *types.ServicePlan) (b bool, e error) {
						return *catalogPlan.Free, nil
					},
					SupportedPlatformsFunc: func(ctx context.Context, plan *types.ServicePlan, repository storage.Repository) (map[string]*types.Platform, error) {
						return service_plans.ResolveSupportedPlatformsForPlans(ctx, []*types.ServicePlan{plan}, repository)
					},
				}).OnTxBefore(interceptors.BrokerUpdateNotificationInterceptorName).Register()

				return nil
			}).Build()
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	AfterEach(func() {
		ctx.CleanupPlatforms()
	})
	BeforeEach(func() {
		c = context.TODO()
		objAfterOp = nil

		// register platform for test tenant
		ctx.RegisterTenantPlatform()

		// register platform for another tenant
		otherTenantPlatform = ctx.RegisterPlatform()
		err := ctx.SMRepository.UpdateLabels(context.Background(), types.PlatformType, otherTenantPlatform.GetID(), types.LabelChanges{
			{
				Operation: types.AddLabelOperation,
				Key:       tenantLabelKey,
				Values:    []string{"anotherTenant"},
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		setAllPlatformsActive(ctx, true, time.Now())
	})

	for _, entry := range entries {
		entry := entry

		getNotifications := func(ids ...string) (*types.Notifications, []string) {
			filters := make([]query.Criterion, 0)
			if len(ids) != 0 {
				filters = append(filters, query.ByField(query.NotInOperator, "id", ids...))
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

				expectedPlatformIDs := entry.ExpectedPlatformIDsFunc(objAfterOp)
				if !slice.StringsAnyEquals(expectedPlatformIDs, notification.PlatformID) {
					continue
				}

				resource := gjson.GetBytes(notification.Payload, "new.resource").Value().(common.Object)
				delete(resource, "updated_at")
				delete(resource, "ready")
				delete(objAfterOp, "updated_at")
				delete(objAfterOp, "ready")
				Expect(resource).To(Equal(objAfterOp))

				actualPayload := gjson.GetBytes(notification.Payload, "new.additional").Raw
				actualPayload = entry.ProcessNotificationPayload(actualPayload)
				expectedPayload := entry.ExpectedAdditionalPayloadFunc(objAfterOp, ctx.SMRepository)
				Expect(actualPayload).To(MatchUnorderedJSON(expectedPayload))
				found = true
				break
			}

			if !found {
				Fail(fmt.Sprintf("Expected to find notification for resource type %s", entry.ResourceType))
			}
		}

		verifyDeletionNotificationCreated := func(objAfterOp common.Object, notificationsAfterOp *types.Notifications, expectedOldPayload string) {
			found := false
			for _, notification := range notificationsAfterOp.Notifications {
				if notification.Type != types.DELETED {
					continue
				}

				oldObjID := gjson.GetBytes(notification.Payload, "old.resource.id").String()
				if oldObjID != objAfterOp["id"] {
					continue
				}

				expectedPlatformIDs := entry.ExpectedPlatformIDsFunc(objAfterOp)
				if !slice.StringsAnyEquals(expectedPlatformIDs, notification.PlatformID) {
					continue
				}

				resource := gjson.GetBytes(notification.Payload, "old.resource").Value().(common.Object)
				delete(resource, "updated_at")
				delete(objAfterOp, "updated_at")
				Expect(resource).To(Equal(objAfterOp))

				actualPayload := gjson.GetBytes(notification.Payload, "old.additional").Raw
				actualPayload = entry.ProcessNotificationPayload(actualPayload)
				Expect(actualPayload).To(MatchUnorderedJSON(expectedOldPayload))

				found = true
				break
			}

			if !found {
				Fail(fmt.Sprintf("Expected to find notification for resource type %s", entry.ResourceType))
			}
		}

		verifyModificationNotificationsCreated := func(objBeforeOp, objAfterOp, update common.Object, notificationsAfterOp *types.Notifications) {
			found := false
			var expectedOldPayload string
			for _, notification := range notificationsAfterOp.Notifications {
				if notification.Type != types.MODIFIED {
					continue
				}

				oldObjID := gjson.GetBytes(notification.Payload, "old.resource.id").String()
				newObjID := gjson.GetBytes(notification.Payload, "new.resource.id").String()

				if oldObjID != objBeforeOp["id"] || newObjID != objAfterOp["id"] {
					continue
				}

				expectedPlatformIDs := entry.ExpectedPlatformIDsFunc(objAfterOp)
				if !slice.StringsAnyEquals(expectedPlatformIDs, notification.PlatformID) {
					continue
				}

				Expect(gjson.GetBytes(notification.Payload, "old.resource.labels").Exists()).To(BeFalse())
				oldResource := gjson.GetBytes(notification.Payload, "old.resource").Value().(common.Object)
				labels := objBeforeOp["labels"]
				delete(objBeforeOp, "labels")
				delete(objBeforeOp, "updated_at")
				delete(oldResource, "updated_at")
				Expect(oldResource).To(Equal(objBeforeOp))
				objBeforeOp["labels"] = labels

				actualOldPayload := gjson.GetBytes(notification.Payload, "old.additional").Raw
				actualOldPayload = entry.ProcessNotificationPayload(actualOldPayload)
				expectedOldPayload = entry.ExpectedAdditionalPayloadFunc(objBeforeOp, ctx.SMRepository)
				Expect(actualOldPayload).To(MatchUnorderedJSON(expectedOldPayload))

				newResource := gjson.GetBytes(notification.Payload, "new.resource").Value().(common.Object)
				labels = objAfterOp["labels"]
				delete(objAfterOp, "labels")
				delete(newResource, "labels")
				delete(objAfterOp, "updated_at")
				delete(newResource, "updated_at")
				Expect(newResource).To(Equal(objAfterOp))
				objAfterOp["labels"] = labels

				actualNewPayload := gjson.GetBytes(notification.Payload, "new.additional").Raw
				actualNewPayload = entry.ProcessNotificationPayload(actualNewPayload)
				expectedNewPayload := entry.ExpectedAdditionalPayloadFunc(objAfterOp, ctx.SMRepository)
				Expect(actualNewPayload).To(MatchUnorderedJSON(expectedNewPayload))

				if labels, ok := update["labels"]; ok {
					labelsJSON := gjson.GetBytes(notification.Payload, "label_changes").Raw
					labelBytes, err := json.Marshal(labels)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(labelsJSON).To(MatchUnorderedJSON(labelBytes))
				}

				found = true
				break
			}

			if !found {
				Fail(fmt.Sprintf("Expected to find notification for resource type %s", entry.ResourceType))
			}

			// when visibility platform_id changes:
			if entry.ExpectedPlatformIDsFunc(objBeforeOp)[0] != entry.ExpectedPlatformIDsFunc(objAfterOp)[0] {
				verifyCreationNotificationCreated(objAfterOp, notificationsAfterOp)
				labels := objBeforeOp["labels"]
				delete(objBeforeOp, "labels")
				verifyDeletionNotificationCreated(objBeforeOp, notificationsAfterOp, expectedOldPayload)
				objBeforeOp["labels"] = labels
			}
		}

		verifyNoNotificationCreatedForType := func(objectType types.ObjectType, notifications *types.Notifications) {
			for _, notification := range notifications.Notifications {
				if notification.Resource == objectType {
					Fail(fmt.Sprintf("Found unexpected notification for object type %s", objectType))
				}
			}
		}

		Context(fmt.Sprintf("ON %s resource CREATE", entry.ResourceType), func() {
			AfterEach(func() {
				entry.ResourceDeleteFunc(objAfterOp)
			})

			When("platforms are active", func() {
				BeforeEach(func() {
					setAllPlatformsActive(ctx, true, time.Now())
				})

				It("also creates a CREATED notification", func() {
					_, ids := getNotifications()
					objAfterOp = entry.ResourceCreateFunc()
					notificationsAfterOp, _ := getNotifications(ids...)

					verifyCreationNotificationCreated(objAfterOp, notificationsAfterOp)
					entry.AdditionalVerificationNotificationsFunc(objAfterOp, ctx.SMRepository, notificationsAfterOp)
				})
			})

			if !entry.ResourceNotifyInactivePlatforms {
				When("platforms are not active but were active recently", func() {
					BeforeEach(func() {
						setAllPlatformsActive(ctx, false, time.Now().Add(-ctx.Config.Storage.Notification.KeepFor/2))
					})

					It("should create a CREATED notification", func() {
						_, ids := getNotifications()
						objAfterOp = entry.ResourceCreateFunc()
						notificationsAfterOp, _ := getNotifications(ids...)

						verifyCreationNotificationCreated(objAfterOp, notificationsAfterOp)
						entry.AdditionalVerificationNotificationsFunc(objAfterOp, ctx.SMRepository, notificationsAfterOp)
					})
				})

				When("platforms are not active and were not active for more than the time we keep notifications for", func() {
					BeforeEach(func() {
						setAllPlatformsActive(ctx, false, time.Now().Add(-ctx.Config.Storage.Notification.KeepFor))
					})

					It(fmt.Sprintf("should not create any notifications for resource type %s", entry.ResourceType), func() {
						_, ids := getNotifications()
						objAfterOp = entry.ResourceCreateFunc()
						notificationsAfterOp, _ := getNotifications(ids...)

						verifyNoNotificationCreatedForType(entry.ResourceType, notificationsAfterOp)
					})
				})
			} else {
				When("platforms are not active and were not active for more than the time we keep notifications for", func() {
					BeforeEach(func() {
						setAllPlatformsActive(ctx, false, time.Now().Add(-ctx.Config.Storage.Notification.KeepFor))
					})

					It("should create a CREATED notification", func() {
						_, ids := getNotifications()
						objAfterOp = entry.ResourceCreateFunc()
						notificationsAfterOp, _ := getNotifications(ids...)

						verifyCreationNotificationCreated(objAfterOp, notificationsAfterOp)
						entry.AdditionalVerificationNotificationsFunc(objAfterOp, ctx.SMRepository, notificationsAfterOp)
					})
				})
			}
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

			if !entry.ResourceNotifyInactivePlatforms {
				When("platforms are not active but were active recently", func() {
					BeforeEach(func() {
						setAllPlatformsActive(ctx, false, time.Now().Add(-ctx.Config.Storage.Notification.KeepFor/2))
					})
					It("should create a DELETED notification", func() {
						_, ids := getNotifications()
						oldPayload := entry.ExpectedAdditionalPayloadFunc(objAfterOp, ctx.SMRepository)

						entry.ResourceDeleteFunc(objAfterOp)
						notificationsAfterOp, _ := getNotifications(ids...)

						verifyDeletionNotificationCreated(objAfterOp, notificationsAfterOp, oldPayload)
					})
				})

				When("platforms are not active and were not active for more than the time we keep notifications for", func() {
					BeforeEach(func() {
						setAllPlatformsActive(ctx, false, time.Now().Add(-ctx.Config.Storage.Notification.KeepFor))
					})
					It(fmt.Sprintf("should not create any notifications for resource type %s", entry.ResourceType), func() {
						_, ids := getNotifications()
						entry.ResourceDeleteFunc(objAfterOp)
						notificationsAfterOp, _ := getNotifications(ids...)

						verifyNoNotificationCreatedForType(entry.ResourceType, notificationsAfterOp)
					})
				})
			} else {
				When("platforms are not active and were not active for more than the time we keep notifications for", func() {
					BeforeEach(func() {
						setAllPlatformsActive(ctx, false, time.Now().Add(-ctx.Config.Storage.Notification.KeepFor))
					})
					It("should create a DELETED notification", func() {
						_, ids := getNotifications()
						oldPayload := entry.ExpectedAdditionalPayloadFunc(objAfterOp, ctx.SMRepository)

						entry.ResourceDeleteFunc(objAfterOp)
						notificationsAfterOp, _ := getNotifications(ids...)

						verifyDeletionNotificationCreated(objAfterOp, notificationsAfterOp, oldPayload)
					})
				})
			}

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

			if !entry.ResourceNotifyInactivePlatforms {
				When("platforms are not active but were active recently", func() {
					BeforeEach(func() {
						setAllPlatformsActive(ctx, false, time.Now().Add(-ctx.Config.Storage.Notification.KeepFor/2))
					})

					DescribeTable("also creates one or more notifications", func(update func() common.Object) {
						_, ids := getNotifications()
						updateBody := update()
						objAfterOp = entry.ResourceUpdateFunc(createdObj, updateBody)
						notificationsAfterOp, _ := getNotifications(ids...)

						verifyModificationNotificationsCreated(createdObj, objAfterOp, updateBody, notificationsAfterOp)

					}, updateOpEntries(entry.ResourceUpdates)...)
				})

				When("platforms are not active and were last active more than the notification keep for duration ago", func() {
					BeforeEach(func() {
						setAllPlatformsActive(ctx, false, time.Now().Add(-ctx.Config.Storage.Notification.KeepFor))
					})

					DescribeTable(fmt.Sprintf("does not create any notifications for resource type %s", entry.ResourceType), func(update func() common.Object) {
						_, ids := getNotifications()
						updateBody := update()
						objAfterOp = entry.ResourceUpdateFunc(createdObj, updateBody)
						notificationsAfterOp, _ := getNotifications(ids...)

						verifyNoNotificationCreatedForType(entry.ResourceType, notificationsAfterOp)
					}, updateOpEntries(entry.ResourceUpdates)...)
				})
			} else {
				When("platforms are not active and were last active more than the notification keep for duration ago", func() {
					BeforeEach(func() {
						setAllPlatformsActive(ctx, false, time.Now().Add(-ctx.Config.Storage.Notification.KeepFor))
					})

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
	}

	Context("When technical platform exists", func() {
		BeforeEach(func() {
			_, err := ctx.SMRepository.Create(context.Background(), &types.Platform{
				Base: types.Base{
					ID: "1234",
				},
				Active:    true,
				Name:      "platform",
				Technical: true,
			})
			Expect(err).ToNot(HaveOccurred())
			regBroker(ctx)
		})

		When("new broker created", func() {
			It("should not create broker notifications", func() {
				c := query.ByField(query.EqualsOperator, "platform_id", "1234")
				cnt, err := ctx.SMRepository.Count(context.Background(), types.NotificationType, c)
				Expect(err).ToNot(HaveOccurred())
				Expect(cnt).To(Equal(0))
			})
		})

		//change this test when feature implemented
		When("new visibility created", func() {
			BeforeEach(func() {
				existingPlanIDs := ctx.SMWithOAuth.List(web.ServicePlansURL).
					Path("$[*].id").Array().Raw()

				v := &types.Visibility{
					Base: types.Base{
						ID:    "111111",
						Ready: true,
					},
					PlatformID:    "1234",
					ServicePlanID: existingPlanIDs[0].(string),
				}
				_, err := ctx.SMRepository.Create(context.Background(), v)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should create visibility notifications", func() {
				c := query.ByField(query.EqualsOperator, "platform_id", "1234")
				cnt, err := ctx.SMRepository.Count(context.Background(), types.NotificationType, c)
				Expect(err).ToNot(HaveOccurred())
				Expect(cnt > 0).To(BeTrue())
			})
		})
	})

	Context("When resource creation fails after the transaction is commited", func() {
		var customCtx *common.TestContext
		BeforeEach(func() {
			customCtx = common.NewTestContextBuilderWithSecurity().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
				smb.WithCreateAroundTxInterceptorProvider(types.ServiceBrokerType, &testCreateInterceptorProvider{}).Register()

				return nil
			}).Build()
		})

		AfterEach(func() {
			customCtx.Cleanup()
		})

		It("should not send create notification", func() {
			list, err := customCtx.SMRepository.List(context.Background(), types.NotificationType, query.ByField(query.EqualsOperator, "type", "CREATED"))
			Expect(err).ShouldNot(HaveOccurred())
			notificationsCountBeforeOp := list.Len()
			regBroker(customCtx)

			brokers, err := customCtx.SMRepository.List(context.Background(), types.ServiceBrokerType)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(brokers.ItemAt(0).GetReady()).To(BeFalse())
			list, err = customCtx.SMRepository.List(context.Background(), types.NotificationType, query.ByField(query.EqualsOperator, "type", "CREATED"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(list.Len()).To(Equal(notificationsCountBeforeOp))

		})
	})
})

func setAllPlatformsActive(ctx *common.TestContext, active bool, lastActive time.Time) {
	platforms, err := ctx.SMRepository.List(context.Background(), types.PlatformType)
	Expect(err).ToNot(HaveOccurred())
	for i := 0; i < platforms.Len(); i++ {
		platform := platforms.ItemAt(i).(*types.Platform)
		setPlatformActive(ctx, platform, active, lastActive)
	}
}

func setPlatformActive(ctx *common.TestContext, platform *types.Platform, active bool, lastActive time.Time) {
	platform.Active = active
	platform.LastActive = lastActive
	_, err := ctx.SMRepository.Update(context.Background(), platform, nil)
	Expect(err).ToNot(HaveOccurred())
}

func regBroker(ctx *common.TestContext) {
	brokerServer := common.NewBrokerServerWithCatalog(common.NewRandomSBCatalog())
	defer brokerServer.Close()

	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	UUID2, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	brokerJSON := common.Object{
		"name":        UUID.String(),
		"broker_url":  brokerServer.URL(),
		"description": UUID2.String(),
		"credentials": common.Object{
			"basic": common.Object{
				"username": brokerServer.Username,
				"password": brokerServer.Password,
			},
		},
	}
	ctx.SMWithOAuth.POST(web.ServiceBrokersURL).
		WithHeaders(map[string]string{}).
		WithJSON(brokerJSON).Expect()
}

type testCreateInterceptorProvider struct {
}

func (p *testCreateInterceptorProvider) Provide() storage.CreateAroundTxInterceptor {
	return &testCreateInterceptor{}
}

func (p *testCreateInterceptorProvider) Name() string {
	return "TestInterceptor"
}

type testCreateInterceptor struct{}

func (p *testCreateInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		robj, err := h(ctx, obj)
		if err != nil {
			return nil, err
		}
		return robj, errors.New("test test")
	}
}
