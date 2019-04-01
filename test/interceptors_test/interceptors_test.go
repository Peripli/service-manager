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

package interceptors_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/extension"
	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/extension/extensionfakes"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInfo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Interceptor Suite")
}

var _ = Describe("Interceptors", func() {
	var ctx *common.TestContext

	var createStack *callStack
	var updateStack *callStack
	var deleteStack *callStack

	var createModificationInterceptor *extensionfakes.FakeCreateInterceptor
	var updateModificationInterceptor *extensionfakes.FakeUpdateInterceptor
	var deleteModificationInterceptor *extensionfakes.FakeDeleteInterceptor

	clearStacks := func() {
		createStack.Clear()
		updateStack.Clear()
		deleteStack.Clear()
	}

	resetModificationInterceptors := func() {
		createModificationInterceptor.OnAPICreateStub = func(h extension.InterceptCreateOnAPI) extension.InterceptCreateOnAPI {
			return h
		}
		createModificationInterceptor.OnTransactionCreateStub = func(f extension.InterceptCreateOnTx) extension.InterceptCreateOnTx {
			return f
		}
		updateModificationInterceptor.OnAPIUpdateStub = func(h extension.InterceptUpdateOnAPI) extension.InterceptUpdateOnAPI {
			return h
		}
		updateModificationInterceptor.OnTransactionUpdateStub = func(f extension.InterceptUpdateOnTx) extension.InterceptUpdateOnTx {
			return f
		}
		deleteModificationInterceptor.OnAPIDeleteStub = func(h extension.InterceptDeleteOnAPI) extension.InterceptDeleteOnAPI {
			return h
		}
		deleteModificationInterceptor.OnTransactionDeleteStub = func(f extension.InterceptDeleteOnTx) extension.InterceptDeleteOnTx {
			return f
		}
	}

	BeforeSuite(func() {
		createStack = &callStack{}
		updateStack = &callStack{}
		deleteStack = &callStack{}

		createModificationInterceptor = &extensionfakes.FakeCreateInterceptor{}
		updateModificationInterceptor = &extensionfakes.FakeUpdateInterceptor{}
		deleteModificationInterceptor = &extensionfakes.FakeDeleteInterceptor{}
		resetModificationInterceptors()

		contextBuilder := common.NewTestContextBuilder()
		contextBuilder.WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			for _, entityType := range []types.ObjectType{types.ServiceBrokerType, types.PlatformType, types.VisibilityType} {
				// Create entity interceptors
				fakeCreateInterceptorProvider0 := createInterceptorProvider(string(entityType)+"0", createStack)
				fakeCreateInterceptorProvider1 := createInterceptorProvider(string(entityType)+"1", createStack)
				fakeCreateInterceptorProvider2 := createInterceptorProvider(string(entityType)+"2", createStack)
				fakeCreateInterceptorProviderBA := createInterceptorProvider(string(entityType)+"APIBefore_TXAfter", createStack)
				fakeCreateInterceptorProviderAB := createInterceptorProvider(string(entityType)+"APIAfter_TXBefore", createStack)

				// Update entity interceptors
				fakeUpdateInterceptorProvider0 := updateInterceptorProvider(string(entityType)+"0", updateStack)
				fakeUpdateInterceptorProvider1 := updateInterceptorProvider(string(entityType)+"1", updateStack)
				fakeUpdateInterceptorProvider2 := updateInterceptorProvider(string(entityType)+"2", updateStack)
				fakeUpdateInterceptorProviderBA := updateInterceptorProvider(string(entityType)+"APIBefore_TXAfter", updateStack)
				fakeUpdateInterceptorProviderAB := updateInterceptorProvider(string(entityType)+"APIAfter_TXBefore", updateStack)

				// Delete entity interceptors
				fakeDeleteInterceptorProvider0 := deleteInterceptorProvider(string(entityType)+"0", deleteStack)
				fakeDeleteInterceptorProvider1 := deleteInterceptorProvider(string(entityType)+"1", deleteStack)
				fakeDeleteInterceptorProvider2 := deleteInterceptorProvider(string(entityType)+"2", deleteStack)
				fakeDeleteInterceptorProviderBA := deleteInterceptorProvider(string(entityType)+"APIBefore_TXAfter", deleteStack)
				fakeDeleteInterceptorProviderAB := deleteInterceptorProvider(string(entityType)+"APIAfter_TXBefore", deleteStack)

				modificationCreateInterceptorProvider := &extensionfakes.FakeCreateInterceptorProvider{}
				modificationCreateInterceptorProvider.NameReturns(string(entityType) + "modificationCreate")
				modificationCreateInterceptorProvider.ProvideReturns(createModificationInterceptor)

				modificationUpdateInterceptorProvider := &extensionfakes.FakeUpdateInterceptorProvider{}
				modificationUpdateInterceptorProvider.NameReturns(string(entityType) + "modificationUpdate")
				modificationUpdateInterceptorProvider.ProvideReturns(updateModificationInterceptor)

				modificationDeleteInterceptorProvider := &extensionfakes.FakeDeleteInterceptorProvider{}
				modificationDeleteInterceptorProvider.NameReturns(string(entityType) + "modificationDelete")
				modificationDeleteInterceptorProvider.ProvideReturns(deleteModificationInterceptor)

				// Register create interceptors
				smb.RegisterCreateInterceptorProvider(entityType, fakeCreateInterceptorProvider1).Apply()
				smb.RegisterCreateInterceptorProvider(entityType, fakeCreateInterceptorProvider2).
					After(fakeCreateInterceptorProvider1.Name()).Apply()
				smb.RegisterCreateInterceptorProvider(entityType, fakeCreateInterceptorProvider0).
					Before(fakeCreateInterceptorProvider1.Name()).Apply()
				smb.RegisterCreateInterceptorProvider(entityType, fakeCreateInterceptorProviderBA).
					APIBefore(fakeCreateInterceptorProvider0.Name()).
					TxAfter(fakeCreateInterceptorProvider2.Name()).
					Apply()
				smb.RegisterCreateInterceptorProvider(entityType, fakeCreateInterceptorProviderAB).
					APIAfter(fakeCreateInterceptorProviderBA.Name()).
					TxBefore(fakeCreateInterceptorProviderBA.Name()).
					Apply()
				// Register update interceptors
				smb.RegisterUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProvider1).Apply()
				smb.RegisterUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProvider2).
					After(fakeUpdateInterceptorProvider1.Name()).Apply()
				smb.RegisterUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProvider0).
					Before(fakeUpdateInterceptorProvider1.Name()).Apply()
				smb.RegisterUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProviderBA).
					APIBefore(fakeUpdateInterceptorProvider0.Name()).
					TxAfter(fakeUpdateInterceptorProvider2.Name()).
					Apply()
				smb.RegisterUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProviderAB).
					APIAfter(fakeUpdateInterceptorProviderBA.Name()).
					TxBefore(fakeUpdateInterceptorProviderBA.Name()).
					Apply()
				// Register delete interceptors
				smb.RegisterDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProvider1).Apply()
				smb.RegisterDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProvider2).
					After(fakeDeleteInterceptorProvider1.Name()).Apply()
				smb.RegisterDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProvider0).
					Before(fakeDeleteInterceptorProvider1.Name()).Apply()
				smb.RegisterDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProviderBA).
					APIBefore(fakeDeleteInterceptorProvider0.Name()).
					TxAfter(fakeDeleteInterceptorProvider2.Name()).
					Apply()
				smb.RegisterDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProviderAB).
					APIAfter(fakeDeleteInterceptorProviderBA.Name()).
					TxBefore(fakeDeleteInterceptorProviderBA.Name()).
					Apply()

				smb.RegisterCreateInterceptorProvider(entityType, modificationCreateInterceptorProvider).Apply()
				smb.RegisterUpdateInterceptorProvider(entityType, modificationUpdateInterceptorProvider).Apply()
				smb.RegisterDeleteInterceptorProvider(entityType, modificationDeleteInterceptorProvider).Apply()
			}

			return nil
		})
		ctx = contextBuilder.Build()
	})

	AfterSuite(func() {
		if ctx != nil {
			ctx.Cleanup()
		}
	})

	BeforeEach(func() {
		clearStacks()
		resetModificationInterceptors()
	})

	AfterEach(func() {
		if ctx != nil {
			ctx.CleanupAdditionalResources()
		}
	})

	Describe("Positioning", func() {
		checkCreateStack := func(objectType types.ObjectType) {
			Expect(createStack.Items).To(Equal(getSequence("Create", objectType)))
			Expect(updateStack.Items).To(HaveLen(0))
			Expect(deleteStack.Items).To(HaveLen(0))
			clearStacks()
		}
		checkUpdateStack := func(objectType types.ObjectType) {
			Expect(createStack.Items).To(HaveLen(0))
			Expect(updateStack.Items).To(Equal(getSequence("Update", objectType)))
			Expect(deleteStack.Items).To(HaveLen(0))
			clearStacks()
		}
		checkDeleteStack := func(objectType types.ObjectType) {
			Expect(createStack.Items).To(HaveLen(0))
			Expect(updateStack.Items).To(HaveLen(0))
			Expect(deleteStack.Items).To(Equal(getSequence("Delete", objectType)))
			clearStacks()
		}

		Context("Broker", func() {
			It("Should call interceptors in right order", func() {
				brokerID, _, _ := ctx.RegisterBrokerWithCatalog(common.NewRandomSBCatalog()) // Post /v1/service_brokers
				checkCreateStack(types.ServiceBrokerType)

				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).WithJSON(common.Object{}).Expect().Status(http.StatusOK)
				checkUpdateStack(types.ServiceBrokerType)

				ctx.CleanupBroker(brokerID) // Delete /v1/service_brokers/<id>
				checkDeleteStack(types.ServiceBrokerType)
			})
		})

		Context("Platform", func() {
			It("Should call interceptors in right order", func() {
				platform := ctx.RegisterPlatform() // Post /v1/platforms
				checkCreateStack(types.PlatformType)

				ctx.SMWithOAuth.PATCH(web.PlatformsURL + "/" + platform.ID).WithJSON(common.Object{}).Expect().Status(http.StatusOK)
				checkUpdateStack(types.PlatformType)

				ctx.SMWithOAuth.DELETE(web.PlatformsURL + "/" + platform.ID).WithJSON(common.Object{}).Expect().Status(http.StatusOK)
				checkDeleteStack(types.PlatformType)
			})
		})

		Context("Visibilities", func() {
			It("Should call interceptors in right order", func() {
				platform := ctx.RegisterPlatform() // Post /v1/platforms
				ctx.RegisterBroker()
				plans := ctx.SMWithBasic.GET(web.ServicePlansURL).Expect().JSON().Object().Value("service_plans").Array()
				planID := plans.First().Object().Value("id").String().Raw()
				clearStacks()
				visibility := types.Visibility{
					PlatformID:    platform.ID,
					ServicePlanID: planID,
				}
				visibilityID := ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(visibility).Expect().
					Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
				checkCreateStack(types.VisibilityType)

				ctx.SMWithOAuth.PATCH(web.VisibilitiesURL + "/" + visibilityID).WithJSON(common.Object{}).Expect().Status(http.StatusOK)
				checkUpdateStack(types.VisibilityType)

				ctx.SMWithOAuth.DELETE(web.VisibilitiesURL + "/" + visibilityID).Expect().Status(http.StatusOK)
				checkDeleteStack(types.VisibilityType)
			})
		})

	})

	Describe("Parameter modification", func() {
		type entry struct {
			createEntryFunc func() string
			url             string
			name            string
		}
		entries := []entry{
			{
				createEntryFunc: func() string {
					brokerID, _, _ := ctx.RegisterBroker()
					return brokerID
				},
				url:  web.ServiceBrokersURL,
				name: string(types.ServiceBrokerType),
			},
			{
				createEntryFunc: func() string {
					platform := ctx.RegisterPlatform()
					return platform.ID
				},
				url:  web.PlatformsURL,
				name: string(types.PlatformType),
			},
			{
				createEntryFunc: func() string {
					platform := ctx.RegisterPlatform() // Post /v1/platforms
					ctx.RegisterBroker()
					plans := ctx.SMWithBasic.GET(web.ServicePlansURL).Expect().JSON().Object().Value("service_plans").Array()
					planID := plans.First().Object().Value("id").String().Raw()
					visibility := types.Visibility{
						PlatformID:    platform.ID,
						ServicePlanID: planID,
					}
					return ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(visibility).Expect().
						Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
				},
				url:  web.VisibilitiesURL,
				name: string(types.VisibilityType),
			},
		}
		for _, e := range entries {
			Context(e.name, func() {
				It("Create", func() {
					newLabelKey := "newLabelKey"
					newLabelValue := []string{"newLabelValueAPI", "newLabelValueTX"}

					createModificationInterceptor.OnAPICreateStub = func(h extension.InterceptCreateOnAPI) extension.InterceptCreateOnAPI {
						return func(ctx context.Context, obj types.Object) (object types.Object, e error) {
							obj.SetLabels(types.Labels{newLabelKey: []string{newLabelValue[0]}})
							obj, e = h(ctx, obj)
							return obj, e
						}
					}
					createModificationInterceptor.OnTransactionCreateStub = func(f extension.InterceptCreateOnTx) extension.InterceptCreateOnTx {
						return func(ctx context.Context, txStorage storage.Warehouse, newObject types.Object) error {
							labels := newObject.GetLabels()
							labels[newLabelKey] = append(labels[newLabelKey], newLabelValue[1])
							newObject.SetLabels(labels)
							return f(ctx, txStorage, newObject)
						}
					}
					entityID := e.createEntryFunc()
					ctx.SMWithOAuth.GET(e.url + "/" + entityID).Expect().
						JSON().Object().
						Value("labels").Object().
						Value(newLabelKey).Array().Equal(newLabelValue)
				})

				It("Update", func() {
					newLabelKey := "newLabelKey"
					newLabelValue := []string{"newLabelValueAPI", "newLabelValueTX"}
					updateModificationInterceptor.OnAPIUpdateStub = func(h extension.InterceptUpdateOnAPI) extension.InterceptUpdateOnAPI {
						return func(ctx context.Context, changes *extension.UpdateContext) (object types.Object, e error) {
							changes.LabelChanges = append(changes.LabelChanges, &query.LabelChange{
								Key:       newLabelKey,
								Operation: query.AddLabelOperation,
								Values:    []string{newLabelValue[0]},
							})
							return h(ctx, changes)
						}
					}
					updateModificationInterceptor.OnTransactionUpdateStub = func(f extension.InterceptUpdateOnTx) extension.InterceptUpdateOnTx {
						return func(ctx context.Context, txStorage storage.Warehouse, oldObject types.Object, changes *extension.UpdateContext) (object types.Object, e error) {
							changes.LabelChanges = append(changes.LabelChanges, &query.LabelChange{
								Key:       newLabelKey,
								Operation: query.AddLabelOperation,
								Values:    []string{newLabelValue[1]},
							})
							return f(ctx, txStorage, oldObject, changes)
						}
					}
					entityID := e.createEntryFunc()
					ctx.SMWithOAuth.PATCH(e.url + "/" + entityID).WithJSON(common.Object{}).
						Expect().JSON().Object().
						Value("labels").Object().
						Value(newLabelKey).Array().Equal(newLabelValue)
				})

				It("Delete", func() {
					_ = e.createEntryFunc()
					deleteModificationInterceptor.OnAPIDeleteStub = func(h extension.InterceptDeleteOnAPI) extension.InterceptDeleteOnAPI {
						return func(ctx context.Context, deletionCriteria ...query.Criterion) (list types.ObjectList, e error) {
							deletionCriteria = append(deletionCriteria, query.ByField(query.InOperator, "id", "invalid"))
							return h(ctx, deletionCriteria...)
						}
					}
					ctx.SMWithOAuth.DELETE(e.url).Expect().Status(http.StatusNotFound)
					resetModificationInterceptors()
					deleteModificationInterceptor.OnTransactionDeleteStub = func(f extension.InterceptDeleteOnTx) extension.InterceptDeleteOnTx {
						return func(ctx context.Context, txStorage storage.Warehouse, deletionCriteria ...query.Criterion) (list types.ObjectList, e error) {
							deletionCriteria = append(deletionCriteria, query.ByField(query.InOperator, "id", "invalid"))
							return f(ctx, txStorage, deletionCriteria...)
						}
					}
					ctx.SMWithOAuth.DELETE(e.url).Expect().Status(http.StatusNotFound)
					resetModificationInterceptors()
					ctx.SMWithOAuth.DELETE(e.url).Expect().Status(http.StatusOK)
				})
			})
		}
	})

})

type callStack struct {
	Items []string
}

func (s *callStack) Add(item string) {
	s.Items = append(s.Items, item)
}

func (s *callStack) Clear() {
	s.Items = make([]string, 0)
}

func createInterceptorProvider(nameSuffix string, stack *callStack) *extensionfakes.FakeCreateInterceptorProvider {
	name := "Create" + nameSuffix
	fakeCreateInterceptorProvider := &extensionfakes.FakeCreateInterceptorProvider{}
	fakeCreateInterceptorProvider.NameReturns(name)

	fakeCreateInterceptor := &extensionfakes.FakeCreateInterceptor{}
	fakeCreateInterceptor.OnAPICreateStub = func(h extension.InterceptCreateOnAPI) extension.InterceptCreateOnAPI {
		return func(ctx context.Context, obj types.Object) (types.Object, error) {
			stack.Add(name + "APIpre")
			obj, err := h(ctx, obj)
			stack.Add(name + "APIpost")
			return obj, err
		}
	}
	fakeCreateInterceptor.OnTransactionCreateStub = func(h extension.InterceptCreateOnTx) extension.InterceptCreateOnTx {
		return func(ctx context.Context, txStorage storage.Warehouse, newObject types.Object) error {
			stack.Add(name + "TXpre")
			err := h(ctx, txStorage, newObject)
			stack.Add(name + "TXpost")
			return err
		}
	}
	fakeCreateInterceptorProvider.ProvideReturns(fakeCreateInterceptor)
	return fakeCreateInterceptorProvider
}

func updateInterceptorProvider(nameSuffix string, stack *callStack) *extensionfakes.FakeUpdateInterceptorProvider {
	name := "Update" + nameSuffix

	fakeUpdateInterceptorProvider := &extensionfakes.FakeUpdateInterceptorProvider{}
	fakeUpdateInterceptorProvider.NameReturns(name)

	fakeUpdateInterceptor := &extensionfakes.FakeUpdateInterceptor{}
	fakeUpdateInterceptor.OnAPIUpdateStub = func(h extension.InterceptUpdateOnAPI) extension.InterceptUpdateOnAPI {
		return func(ctx context.Context, changes *extension.UpdateContext) (types.Object, error) {
			stack.Add(name + "APIpre")
			obj, err := h(ctx, changes)
			stack.Add(name + "APIpost")
			return obj, err
		}
	}
	fakeUpdateInterceptor.OnTransactionUpdateStub = func(h extension.InterceptUpdateOnTx) extension.InterceptUpdateOnTx {
		return func(ctx context.Context, txStorage storage.Warehouse, oldObject types.Object, changes *extension.UpdateContext) (types.Object, error) {
			stack.Add(name + "TXpre")
			obj, err := h(ctx, txStorage, oldObject, changes)
			stack.Add(name + "TXpost")
			return obj, err
		}
	}
	fakeUpdateInterceptorProvider.ProvideReturns(fakeUpdateInterceptor)
	return fakeUpdateInterceptorProvider
}

func deleteInterceptorProvider(nameSuffix string, stack *callStack) *extensionfakes.FakeDeleteInterceptorProvider {
	name := "Delete" + nameSuffix

	fakeDeleteInterceptorProvider := &extensionfakes.FakeDeleteInterceptorProvider{}
	fakeDeleteInterceptorProvider.NameReturns(name)

	fakeDeleteInterceptor := &extensionfakes.FakeDeleteInterceptor{}
	fakeDeleteInterceptor.OnAPIDeleteStub = func(h extension.InterceptDeleteOnAPI) extension.InterceptDeleteOnAPI {
		return func(ctx context.Context, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
			stack.Add(name + "APIpre")
			obj, err := h(ctx, deletionCriteria...)
			stack.Add(name + "APIpost")
			return obj, err
		}
	}
	fakeDeleteInterceptor.OnTransactionDeleteStub = func(h extension.InterceptDeleteOnTx) extension.InterceptDeleteOnTx {
		return func(ctx context.Context, txStorage storage.Warehouse, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
			stack.Add(name + "TXpre")
			obj, err := h(ctx, txStorage, deletionCriteria...)
			stack.Add(name + "TXpost")
			return obj, err
		}
	}
	fakeDeleteInterceptorProvider.ProvideReturns(fakeDeleteInterceptor)
	return fakeDeleteInterceptorProvider
}

func getSequence(operation string, objectType types.ObjectType) []string {
	sequence := []string{"APIBefore_TXAfterAPIpre", "APIAfter_TXBeforeAPIpre", "0APIpre", "1APIpre", "2APIpre",
		"0TXpre", "1TXpre", "2TXpre", "APIAfter_TXBeforeTXpre", "APIBefore_TXAfterTXpre",
		"APIBefore_TXAfterTXpost", "APIAfter_TXBeforeTXpost", "2TXpost", "1TXpost", "0TXpost",
		"2APIpost", "1APIpost", "0APIpost", "APIAfter_TXBeforeAPIpost", "APIBefore_TXAfterAPIpost"}
	prefixedSequence := make([]string, 0, len(sequence))
	for _, value := range sequence {
		prefixedSequence = append(prefixedSequence, operation+string(objectType)+value)
	}
	return prefixedSequence
}
