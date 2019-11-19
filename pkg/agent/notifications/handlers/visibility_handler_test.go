package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Peripli/service-manager/test/common"

	"github.com/tidwall/gjson"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/agent/platform"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/pkg/agent/notifications/handlers"
	"github.com/Peripli/service-manager/pkg/agent/platform/platformfakes"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("Visibility Handler", func() {
	var ctx context.Context

	var fakeVisibilityClient *platformfakes.FakeVisibilityClient

	var visibilityHandler *handlers.VisibilityResourceNotificationsHandler

	var visibilityNotificationPayload string

	var labels string
	var catalogPlan string
	var catalogPlanID string
	var anotherCatalogPlanID string
	var smBrokerID string
	var smBrokerName string

	var err error

	unmarshalLabels := func(labelsJSON string) types.Labels {
		labels := types.Labels{}
		err := json.Unmarshal([]byte(labelsJSON), &labels)
		Expect(err).ShouldNot(HaveOccurred())

		return labels
	}

	unmarshalLabelChanges := func(labelChangeJSON string) query.LabelChanges {
		labelChanges := query.LabelChanges{}
		err := json.Unmarshal([]byte(labelChangeJSON), &labelChanges)
		Expect(err).ShouldNot(HaveOccurred())

		return labelChanges
	}

	BeforeEach(func() {
		ctx = context.TODO()

		smBrokerID = "brokerID"
		smBrokerName = "brokerName"

		labels = `
		{
			"key1": ["value1", "value2"],
			"key2": ["value3", "value4"]
		}`
		catalogPlan = `
		{
			"name": "another-plan-name-123",
			"id": "another-plan-id-123",
			"catalog_id": "catalog_id-123",
			"catalog_name": "catalog_name-123",
			"service_offering_id": "so-id-123",
			"description": "test-description",
			"free": true,
			"metadata": {
				"max_storage_tb": 5,
				"costs":[
					{
						"amount":{
							"usd":199.0
						},
						"unit":"MONTHLY"
					},
					{
				   		"amount":{
					  		"usd":0.99
				   		},
				   		"unit":"1GB of messages over 20GB"
					}
			 	],
				"bullets": [
			  	"40 concurrent connections"
				]
		  	}
		}`

		catalogPlanID = gjson.Get(catalogPlan, "catalog_id").Str
		anotherCatalogPlanID = "anotherCatalogPlanID"

		fakeVisibilityClient = &platformfakes.FakeVisibilityClient{}

		visibilityHandler = &handlers.VisibilityResourceNotificationsHandler{
			VisibilityClient: fakeVisibilityClient,
		}
	})

	Describe("OnCreate", func() {
		BeforeEach(func() {
			visibilityNotificationPayload = fmt.Sprintf(`
			{
				"new": {
					"resource": {
						"id": "visID",
						"platform_id": "smPlatformID",
						"service_plan_id": "smServicePlanID",
						"labels": %s
					},
					"additional": {
						"broker_id": "%s",
						"broker_name": "%s",
						"service_plan": %s
					}
				}
			}`, labels, smBrokerID, smBrokerName, catalogPlan)
		})

		Context("when visibility client is nil", func() {
			It("does not try to enable or disable access", func() {
				h := &handlers.VisibilityResourceNotificationsHandler{}
				h.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				visibilityNotificationPayload = `randomString`
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when notification payload is invalid", func() {
			Context("when new resource is missing", func() {
				BeforeEach(func() {
					visibilityNotificationPayload = `{"randomKey":"randomValue"}`
				})

				It("does not try to enable or disable access", func() {
					visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
				})
			})

			Context("when the visibility ID is missing", func() {
				BeforeEach(func() {
					visibilityNotificationPayload, err = sjson.Delete(visibilityNotificationPayload, "new.resource.id")
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("does not try to enable or disable access", func() {
					visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
				})
			})

			Context("when the visibility plan ID is missing", func() {
				BeforeEach(func() {
					visibilityNotificationPayload, err = sjson.Delete(visibilityNotificationPayload, "new.resource.service_plan_id")
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("does not try to enable or disable access", func() {
					visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
				})
			})

			Context("when the catalog plan is missing", func() {
				BeforeEach(func() {
					visibilityNotificationPayload, err = sjson.Delete(visibilityNotificationPayload, "new.additional.service_plan")
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("does not try to enable or disable access", func() {
					visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
				})
			})
		})

		Context("when the broker ID is missing", func() {
			BeforeEach(func() {
				visibilityNotificationPayload, err = sjson.Delete(visibilityNotificationPayload, "new.additional.broker_id")
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when the broker name is missing", func() {
			BeforeEach(func() {
				visibilityNotificationPayload, err = sjson.Delete(visibilityNotificationPayload, "new.additional.broker_name")
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when the visibility notification is for a broker which is in the broker blacklist", func() {
			It("does not try to enable or disable access", func() {
				visibilityHandler.BrokerBlacklist = []string{smBrokerName}
				visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when the notification payload is valid", func() {
			Context("when an error occurs while enabling access", func() {
				BeforeEach(func() {
					fakeVisibilityClient.EnableAccessForPlanReturns(fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedRequest *platform.ModifyPlanAccessRequest

				BeforeEach(func() {
					fakeVisibilityClient.EnableAccessForPlanReturns(nil)

					expectedRequest = &platform.ModifyPlanAccessRequest{
						BrokerName:    brokerProxyName(visibilityHandler.ProxyPrefix, smBrokerName, smBrokerID),
						CatalogPlanID: catalogPlanID,
						Labels:        unmarshalLabels(labels),
					}
				})

				It("invokes enable access for plan", func() {
					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(1))

					callCtx, callRequest := fakeVisibilityClient.EnableAccessForPlanArgsForCall(0)

					Expect(callCtx).To(Equal(ctx))
					Expect(callRequest).To(Equal(expectedRequest))
				})
			})
		})
	})

	Describe("OnUpdate", func() {
		var addLabelChanges string
		var removeLabelChanges string
		var labelChanges string

		BeforeEach(func() {
			addLabelChanges = `
			{
				"op": "add",
				"key": "key3",
				"values": ["value5", "value6"]
			}`

			removeLabelChanges = `
			{
				"op": "remove",
				"key": "key2",
				"values": ["value3", "value4"]
			}`

			labelChanges = fmt.Sprintf(`
			[%s, %s]`, addLabelChanges, removeLabelChanges)

			visibilityNotificationPayload = fmt.Sprintf(`
			{
				"old": {
					"resource": {
						"id": "visID",
						"platform_id": "smPlatformID",
						"service_plan_id": "smServicePlanID",
						"labels": %s
					},
					"additional": {
						"broker_id": "%s",
						"broker_name": "%s",
						"service_plan": %s
					}
				},
				"new": {
					"resource": {
						"id": "visID",
						"platform_id": "smPlatformID",
						"service_plan_id": "smServicePlanID",
						"labels": %s
					},
					"additional": {
						"broker_id": "%s",
						"broker_name": "%s",
						"service_plan": %s
					}
				},
				"label_changes": %s
			}`, labels, smBrokerID, smBrokerName, catalogPlan, labels, smBrokerID, smBrokerName, catalogPlan, labelChanges)
		})

		Context("when visibility client is nil", func() {
			It("does not try to enable or disable access", func() {
				h := &handlers.VisibilityResourceNotificationsHandler{}
				h.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				visibilityNotificationPayload = `randomString`
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when old resource is missing", func() {
			BeforeEach(func() {
				visibilityNotificationPayload, err = sjson.Delete(visibilityNotificationPayload, "old")
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when new resource is missing", func() {
			BeforeEach(func() {
				visibilityNotificationPayload, err = sjson.Delete(visibilityNotificationPayload, "new")
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when the visibility notification is for a broker which is in the broker blacklist", func() {
			It("does not try to enable or disable access", func() {
				visibilityHandler.BrokerBlacklist = []string{smBrokerName}
				visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when the notification payload is valid", func() {
			Context("when an error occurs while enabling access", func() {
				BeforeEach(func() {
					fakeVisibilityClient.EnableAccessForPlanReturns(fmt.Errorf("error"))
					fakeVisibilityClient.DisableAccessForPlanReturns(nil)

				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))
					})
				})
			})

			Context("when an error occurs while disabling access", func() {
				BeforeEach(func() {
					fakeVisibilityClient.EnableAccessForPlanReturns(nil)
					fakeVisibilityClient.DisableAccessForPlanReturns(fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedEnableAccessRequests []*platform.ModifyPlanAccessRequest
				var expectedDisableAccessRequests []*platform.ModifyPlanAccessRequest
				var labelsToAdd types.Labels
				var labelsToRemove types.Labels

				verifyInvocations := func(enableAccessCount, disableAccessCount int) {
					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))

					visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(enableAccessCount))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(disableAccessCount))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(len(expectedEnableAccessRequests)))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(len(expectedDisableAccessRequests)))

					for i, expectedRequest := range expectedEnableAccessRequests {
						callCtx, enableAccessRequest := fakeVisibilityClient.EnableAccessForPlanArgsForCall(i)
						Expect(callCtx).To(Equal(ctx))
						Expect(enableAccessRequest).To(Equal(expectedRequest))
					}

					for i, expectedRequest := range expectedDisableAccessRequests {
						callCtx, enableAccessRequest := fakeVisibilityClient.DisableAccessForPlanArgsForCall(i)
						Expect(callCtx).To(Equal(ctx))
						Expect(enableAccessRequest).To(Equal(expectedRequest))
					}
				}

				BeforeEach(func() {
					fakeVisibilityClient.EnableAccessForPlanReturns(nil)
					fakeVisibilityClient.DisableAccessForPlanReturns(nil)

					labelsToAdd, labelsToRemove = handlers.LabelChangesToLabels(unmarshalLabelChanges(labelChanges))
					expectedEnableAccessRequests = []*platform.ModifyPlanAccessRequest{
						{
							BrokerName:    brokerProxyName(visibilityHandler.ProxyPrefix, smBrokerName, smBrokerID),
							CatalogPlanID: catalogPlanID,
							Labels:        labelsToAdd,
						},
					}

					expectedDisableAccessRequests = []*platform.ModifyPlanAccessRequest{
						{
							BrokerName:    brokerProxyName(visibilityHandler.ProxyPrefix, smBrokerName, smBrokerID),
							CatalogPlanID: catalogPlanID,
							Labels:        labelsToRemove,
						},
					}
				})

				Context("When the labels are empty and the platform id is not", func() {
					BeforeEach(func() {
						visibilityNotificationPayload, err = sjson.Set(visibilityNotificationPayload, "label_changes", common.Array{})
						Expect(err).ShouldNot(HaveOccurred())
						expectedEnableAccessRequests = []*platform.ModifyPlanAccessRequest{}
						expectedDisableAccessRequests = []*platform.ModifyPlanAccessRequest{}
					})

					It("Does not enable or disable access", func() {
						verifyInvocations(0, 0)
					})
				})

				Context("When the labels are empty and the platform id is empty", func() {
					BeforeEach(func() {
						visibilityNotificationPayload, err = sjson.Set(visibilityNotificationPayload, "new.resource.platform_id", "")
						Expect(err).ShouldNot(HaveOccurred())
						visibilityNotificationPayload, err = sjson.Set(visibilityNotificationPayload, "label_changes", common.Array{})
						Expect(err).ShouldNot(HaveOccurred())
						expectedEnableAccessRequests = []*platform.ModifyPlanAccessRequest{
							{
								BrokerName:    brokerProxyName(visibilityHandler.ProxyPrefix, smBrokerName, smBrokerID),
								CatalogPlanID: catalogPlanID,
								Labels:        types.Labels{},
							},
						}
						expectedDisableAccessRequests = []*platform.ModifyPlanAccessRequest{
							{
								BrokerName:    brokerProxyName(visibilityHandler.ProxyPrefix, smBrokerName, smBrokerID),
								CatalogPlanID: catalogPlanID,
								Labels:        types.Labels{},
							},
						}
					})

					It("Enable and disable access", func() {
						verifyInvocations(1, 1)
					})
				})

				Context("When the labels are not empty and the platform id is not empty", func() {
					BeforeEach(func() {
						visibilityNotificationPayload, err = sjson.Set(visibilityNotificationPayload, "label_changes", common.Array{})
						Expect(err).ShouldNot(HaveOccurred())
						expectedEnableAccessRequests = []*platform.ModifyPlanAccessRequest{}
						expectedDisableAccessRequests = []*platform.ModifyPlanAccessRequest{}
					})

					It("Does not enable or disable access", func() {
						verifyInvocations(0, 0)
					})
				})

				Context("When the labels are not empty and the platform id is not empty", func() {
					It("invokes enable and disable access for the plan", func() {
						verifyInvocations(1, 1)
					})
				})

				Context("when the catalog plan ID is modified", func() {
					BeforeEach(func() {
						visibilityNotificationPayload, err = sjson.Set(visibilityNotificationPayload, "new.additional.service_plan.catalog_id", anotherCatalogPlanID)
						Expect(err).ShouldNot(HaveOccurred())

						expectedEnableAccessRequests = []*platform.ModifyPlanAccessRequest{
							{
								BrokerName:    brokerProxyName(visibilityHandler.ProxyPrefix, smBrokerName, smBrokerID),
								CatalogPlanID: anotherCatalogPlanID,
								Labels:        unmarshalLabels(labels),
							},
							{
								BrokerName:    brokerProxyName(visibilityHandler.ProxyPrefix, smBrokerName, smBrokerID),
								CatalogPlanID: anotherCatalogPlanID,
								Labels:        labelsToAdd,
							},
						}

						expectedDisableAccessRequests = []*platform.ModifyPlanAccessRequest{
							{
								BrokerName:    brokerProxyName(visibilityHandler.ProxyPrefix, smBrokerName, smBrokerID),
								CatalogPlanID: catalogPlanID,
								Labels:        unmarshalLabels(labels),
							},
							{
								BrokerName:    brokerProxyName(visibilityHandler.ProxyPrefix, smBrokerName, smBrokerID),
								CatalogPlanID: anotherCatalogPlanID,
								Labels:        labelsToRemove,
							},
						}
					})

					It("invokes enable access for the new catalog plan id and disable access for the old catalog plan id", func() {
						verifyInvocations(2, 2)
					})
				})
			})
		})
	})

	Describe("OnDelete", func() {
		BeforeEach(func() {
			visibilityNotificationPayload = fmt.Sprintf(`
					{
						"old": {
							"resource": {
								"id": "visID",
								"platform_id": "smPlatformID",
								"service_plan_id": "smServicePlanID",
								"labels": %s
							},
							"additional": {
								"broker_id": "%s",
								"broker_name": "%s",
								"service_plan": %s
							}
						}
					}`, labels, smBrokerID, smBrokerName, catalogPlan)
		})

		Context("when visibility client is nil", func() {
			It("does not try to enable or disable access", func() {
				h := &handlers.VisibilityResourceNotificationsHandler{}
				h.OnDelete(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				visibilityNotificationPayload = `randomString`
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnDelete(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when notification payload is invalid", func() {
			Context("when old resource is missing", func() {
				BeforeEach(func() {
					visibilityNotificationPayload, err = sjson.Delete(visibilityNotificationPayload, "old")
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("does not try to enable or disable access", func() {
					visibilityHandler.OnDelete(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
				})
			})
		})

		Context("when the visibility notification is for a broker which is in the broker blacklist", func() {
			It("does not try to enable or disable access", func() {
				visibilityHandler.BrokerBlacklist = []string{smBrokerName}
				visibilityHandler.OnDelete(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when the notification payload is valid", func() {
			Context("when an error occurs while disabling access", func() {
				BeforeEach(func() {
					fakeVisibilityClient.DisableAccessForPlanReturns(fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						visibilityHandler.OnDelete(ctx, json.RawMessage(visibilityNotificationPayload))
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedRequest *platform.ModifyPlanAccessRequest

				BeforeEach(func() {
					fakeVisibilityClient.DisableAccessForPlanReturns(nil)

					expectedRequest = &platform.ModifyPlanAccessRequest{
						BrokerName:    brokerProxyName(visibilityHandler.ProxyPrefix, smBrokerName, smBrokerID),
						CatalogPlanID: catalogPlanID,
						Labels:        unmarshalLabels(labels),
					}
				})

				It("invokes disable access for plan", func() {
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
					visibilityHandler.OnDelete(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(1))

					callCtx, callRequest := fakeVisibilityClient.DisableAccessForPlanArgsForCall(0)

					Expect(callCtx).To(Equal(ctx))
					Expect(callRequest).To(Equal(expectedRequest))
				})
			})

		})
	})
})
