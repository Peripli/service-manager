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

	"github.com/Peripli/service-manager/storage/storagefakes"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInterceptors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Interceptor Suite")
}

var _ = Describe("Interceptors", func() {
	var ctx *common.TestContext

	var createStack *callStack
	var updateStack *callStack
	var deleteStack *callStack

	var createModificationInterceptors map[types.ObjectType]*storagefakes.FakeCreateInterceptor
	var updateModificationInterceptors map[types.ObjectType]*storagefakes.FakeUpdateInterceptor
	var deleteModificationInterceptors map[types.ObjectType]*storagefakes.FakeDeleteInterceptor

	clearStacks := func() {
		createStack.Clear()
		updateStack.Clear()
		deleteStack.Clear()
	}

	resetModificationInterceptors := func() {

		for _, interceptor := range createModificationInterceptors {
			interceptor.AroundTxCreateCalls(func(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
				return h
			})

			interceptor.OnTxCreateCalls(func(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
				return f
			})
		}

		for _, interceptor := range updateModificationInterceptors {
			interceptor.AroundTxUpdateCalls(func(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
				return h
			})
			interceptor.OnTxUpdateCalls(func(f storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
				return f
			})
		}

		for _, interceptor := range deleteModificationInterceptors {
			interceptor.AroundTxDeleteCalls(func(h storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
				return h
			})
			interceptor.OnTxDeleteCalls(func(f storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
				return f
			})
		}
	}

	BeforeSuite(func() {
		createModificationInterceptors = make(map[types.ObjectType]*storagefakes.FakeCreateInterceptor)
		updateModificationInterceptors = make(map[types.ObjectType]*storagefakes.FakeUpdateInterceptor)
		deleteModificationInterceptors = make(map[types.ObjectType]*storagefakes.FakeDeleteInterceptor)

		createStack = &callStack{}
		updateStack = &callStack{}
		deleteStack = &callStack{}

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

				createModificationInterceptor := &storagefakes.FakeCreateInterceptor{}

				modificationCreateInterceptorProvider := &storagefakes.FakeCreateInterceptorProvider{}
				createModificationInterceptor.NameReturns(string(entityType) + "modificationCreate")
				modificationCreateInterceptorProvider.ProvideReturns(createModificationInterceptor)

				createModificationInterceptors[entityType] = createModificationInterceptor

				updateModificationInterceptor := &storagefakes.FakeUpdateInterceptor{}

				modificationUpdateInterceptorProvider := &storagefakes.FakeUpdateInterceptorProvider{}
				updateModificationInterceptor.NameReturns(string(entityType) + "modificationUpdate")
				modificationUpdateInterceptorProvider.ProvideReturns(updateModificationInterceptor)

				updateModificationInterceptors[entityType] = updateModificationInterceptor

				deleteModificationInterceptor := &storagefakes.FakeDeleteInterceptor{}

				modificationDeleteInterceptorProvider := &storagefakes.FakeDeleteInterceptorProvider{}
				deleteModificationInterceptor.NameReturns(string(entityType) + "modificationDelete")
				modificationDeleteInterceptorProvider.ProvideReturns(deleteModificationInterceptor)

				deleteModificationInterceptors[entityType] = deleteModificationInterceptor

				resetModificationInterceptors()

				// Register create interceptors
				smb.RegisterCreateInterceptorProvider(entityType, fakeCreateInterceptorProvider1).Apply()
				smb.RegisterCreateInterceptorProvider(entityType, fakeCreateInterceptorProvider2).
					After(fakeCreateInterceptorProvider1.Provide().Name()).Apply()
				smb.RegisterCreateInterceptorProvider(entityType, fakeCreateInterceptorProvider0).
					Before(fakeCreateInterceptorProvider1.Provide().Name()).Apply()
				smb.RegisterCreateInterceptorProvider(entityType, fakeCreateInterceptorProviderBA).
					AroundTxBefore(fakeCreateInterceptorProvider0.Provide().Name()).
					TxAfter(fakeCreateInterceptorProvider2.Provide().Name()).
					Apply()
				smb.RegisterCreateInterceptorProvider(entityType, fakeCreateInterceptorProviderAB).
					AroundTxAfter(fakeCreateInterceptorProviderBA.Provide().Name()).
					TxBefore(fakeCreateInterceptorProviderBA.Provide().Name()).
					Apply()
				// Register update interceptors
				smb.RegisterUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProvider1).Apply()
				smb.RegisterUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProvider2).
					After(fakeUpdateInterceptorProvider1.Provide().Name()).Apply()
				smb.RegisterUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProvider0).
					Before(fakeUpdateInterceptorProvider1.Provide().Name()).Apply()
				smb.RegisterUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProviderBA).
					AroundTxBefore(fakeUpdateInterceptorProvider0.Provide().Name()).
					TxAfter(fakeUpdateInterceptorProvider2.Provide().Name()).
					Apply()
				smb.RegisterUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProviderAB).
					AroundTxAfter(fakeUpdateInterceptorProviderBA.Provide().Name()).
					TxBefore(fakeUpdateInterceptorProviderBA.Provide().Name()).
					Apply()
				// Register delete interceptors
				smb.RegisterDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProvider1).Apply()
				smb.RegisterDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProvider2).
					After(fakeDeleteInterceptorProvider1.Provide().Name()).Apply()
				smb.RegisterDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProvider0).
					Before(fakeDeleteInterceptorProvider1.Provide().Name()).Apply()
				smb.RegisterDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProviderBA).
					AroundTxBefore(fakeDeleteInterceptorProvider0.Provide().Name()).
					TxAfter(fakeDeleteInterceptorProvider2.Provide().Name()).
					Apply()
				smb.RegisterDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProviderAB).
					AroundTxAfter(fakeDeleteInterceptorProviderBA.Provide().Name()).
					TxBefore(fakeDeleteInterceptorProviderBA.Provide().Name()).
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
			//{
			//	createEntryFunc: func() string {
			//		brokerID, _, _ := ctx.RegisterBroker()
			//		return brokerID
			//	},
			//	url:  web.ServiceBrokersURL,
			//	name: string(types.ServiceBrokerType),
			//},
			{
				createEntryFunc: func() string {
					platform := ctx.RegisterPlatform()
					return platform.ID
				},
				url:  web.PlatformsURL,
				name: string(types.PlatformType),
			},
			//{
			//	createEntryFunc: func() string {
			//		platform := ctx.RegisterPlatform() // Post /v1/platforms
			//		ctx.RegisterBroker()
			//		plans := ctx.SMWithBasic.GET(web.ServicePlansURL).Expect().JSON().Object().Value("service_plans").Array()
			//		planID := plans.First().Object().Value("id").String().Raw()
			//		visibility := types.Visibility{
			//			PlatformID:    platform.ID,
			//			ServicePlanID: planID,
			//		}
			//		return ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(visibility).Expect().
			//			Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
			//	},
			//	url:  web.VisibilitiesURL,
			//	name: string(types.VisibilityType),
			//},
		}
		for _, e := range entries {
			Context(e.name, func() {
				It("Create", func() {
					newLabelKey := "newLabelKey"
					newLabelValue := []string{"newLabelValueAPI", "newLabelValueTX"}

					createModificationInterceptors[types.ObjectType(e.name)].AroundTxCreateStub = func(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
						return func(ctx context.Context, obj types.Object) (object types.Object, e error) {
							obj.SetLabels(types.Labels{newLabelKey: []string{newLabelValue[0]}})
							obj, e = h(ctx, obj)
							return obj, e
						}
					}
					createModificationInterceptors[types.ObjectType(e.name)].OnTxCreateStub = func(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
						return func(ctx context.Context, txStorage storage.Repository, newObject types.Object) error {
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
					updateModificationInterceptors[types.ObjectType(e.name)].AroundTxUpdateStub = func(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
						return func(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (object types.Object, e error) {
							labelChanges = append(labelChanges, &query.LabelChange{
								Key:       newLabelKey,
								Operation: query.AddLabelOperation,
								Values:    []string{newLabelValue[0]},
							})
							return h(ctx, obj, labelChanges...)
						}
					}
					updateModificationInterceptors[types.ObjectType(e.name)].OnTxUpdateStub = func(f storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
						return func(ctx context.Context, txStorage storage.Repository, obj types.Object, labelChanges ...*query.LabelChange) (object types.Object, e error) {
							labelChanges = append(labelChanges, &query.LabelChange{
								Key:       newLabelKey,
								Operation: query.AddLabelOperation,
								Values:    []string{newLabelValue[1]},
							})
							return f(ctx, txStorage, obj, labelChanges...)
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
					deleteModificationInterceptors[types.ObjectType(e.name)].AroundTxDeleteStub = func(h storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
						return func(ctx context.Context, deletionCriteria ...query.Criterion) (list types.ObjectList, e error) {
							deletionCriteria = append(deletionCriteria, query.ByField(query.InOperator, "id", "invalid"))
							return h(ctx, deletionCriteria...)
						}
					}
					ctx.SMWithOAuth.DELETE(e.url).Expect().Status(http.StatusNotFound)
					resetModificationInterceptors()
					deleteModificationInterceptors[types.ObjectType(e.name)].OnTxDeleteStub = func(f storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
						return func(ctx context.Context, txStorage storage.Repository, deletionCriteria ...query.Criterion) (list types.ObjectList, e error) {
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

func createInterceptorProvider(nameSuffix string, stack *callStack) *storagefakes.FakeCreateInterceptorProvider {
	name := "Create" + nameSuffix
	fakeCreateInterceptorProvider := &storagefakes.FakeCreateInterceptorProvider{}

	fakeCreateInterceptor := &storagefakes.FakeCreateInterceptor{}
	fakeCreateInterceptor.NameReturns(name)
	fakeCreateInterceptor.AroundTxCreateStub = func(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
		return func(ctx context.Context, obj types.Object) (types.Object, error) {
			stack.Add(name + "APIpre")
			obj, err := h(ctx, obj)
			stack.Add(name + "APIpost")
			return obj, err
		}
	}
	fakeCreateInterceptor.OnTxCreateStub = func(h storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
		return func(ctx context.Context, txStorage storage.Repository, newObject types.Object) error {
			stack.Add(name + "TXpre")
			err := h(ctx, txStorage, newObject)
			stack.Add(name + "TXpost")
			return err
		}
	}
	fakeCreateInterceptorProvider.ProvideReturns(fakeCreateInterceptor)
	return fakeCreateInterceptorProvider
}

func updateInterceptorProvider(nameSuffix string, stack *callStack) *storagefakes.FakeUpdateInterceptorProvider {
	name := "Update" + nameSuffix

	fakeUpdateInterceptorProvider := &storagefakes.FakeUpdateInterceptorProvider{}
	fakeUpdateInterceptor := &storagefakes.FakeUpdateInterceptor{}
	fakeUpdateInterceptor.NameReturns(name)
	fakeUpdateInterceptor.AroundTxUpdateStub = func(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
		return func(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
			stack.Add(name + "APIpre")
			obj, err := h(ctx, obj, labelChanges...)
			stack.Add(name + "APIpost")
			return obj, err
		}
	}
	fakeUpdateInterceptor.OnTxUpdateStub = func(h storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
		return func(ctx context.Context, txStorage storage.Repository, object types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
			stack.Add(name + "TXpre")
			obj, err := h(ctx, txStorage, object, labelChanges...)
			stack.Add(name + "TXpost")
			return obj, err
		}
	}
	fakeUpdateInterceptorProvider.ProvideReturns(fakeUpdateInterceptor)
	return fakeUpdateInterceptorProvider
}

func deleteInterceptorProvider(nameSuffix string, stack *callStack) *storagefakes.FakeDeleteInterceptorProvider {
	name := "Delete" + nameSuffix

	fakeDeleteInterceptorProvider := &storagefakes.FakeDeleteInterceptorProvider{}
	fakeDeleteInterceptor := &storagefakes.FakeDeleteInterceptor{}
	fakeDeleteInterceptor.NameReturns(name)
	fakeDeleteInterceptor.AroundTxDeleteStub = func(h storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
		return func(ctx context.Context, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
			stack.Add(name + "APIpre")
			obj, err := h(ctx, deletionCriteria...)
			stack.Add(name + "APIpost")
			return obj, err
		}
	}
	fakeDeleteInterceptor.OnTxDeleteStub = func(h storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
		return func(ctx context.Context, txStorage storage.Repository, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
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
