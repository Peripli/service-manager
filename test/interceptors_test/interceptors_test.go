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
	"errors"
	"fmt"
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
	. "github.com/onsi/ginkgo/v2"
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

		contextBuilder := common.NewTestContextBuilderWithSecurity()
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

				// DeleteReturning entity interceptors
				fakeDeleteInterceptorProvider0 := deleteInterceptorProvider(string(entityType)+"0", deleteStack)
				fakeDeleteInterceptorProvider1 := deleteInterceptorProvider(string(entityType)+"1", deleteStack)
				fakeDeleteInterceptorProvider2 := deleteInterceptorProvider(string(entityType)+"2", deleteStack)
				fakeDeleteInterceptorProviderBA := deleteInterceptorProvider(string(entityType)+"APIBefore_TXAfter", deleteStack)
				fakeDeleteInterceptorProviderAB := deleteInterceptorProvider(string(entityType)+"APIAfter_TXBefore", deleteStack)

				createModificationInterceptor := &storagefakes.FakeCreateInterceptor{}

				modificationCreateInterceptorProvider := &storagefakes.FakeCreateInterceptorProvider{}
				modificationCreateInterceptorProvider.NameReturns(string(entityType) + "modificationCreate")
				modificationCreateInterceptorProvider.ProvideReturns(createModificationInterceptor)

				createModificationInterceptors[entityType] = createModificationInterceptor

				updateModificationInterceptor := &storagefakes.FakeUpdateInterceptor{}

				modificationUpdateInterceptorProvider := &storagefakes.FakeUpdateInterceptorProvider{}
				modificationUpdateInterceptorProvider.NameReturns(string(entityType) + "modificationUpdate")
				modificationUpdateInterceptorProvider.ProvideReturns(updateModificationInterceptor)

				updateModificationInterceptors[entityType] = updateModificationInterceptor

				deleteModificationInterceptor := &storagefakes.FakeDeleteInterceptor{}

				modificationDeleteInterceptorProvider := &storagefakes.FakeDeleteInterceptorProvider{}
				modificationDeleteInterceptorProvider.NameReturns(string(entityType) + "modificationDelete")
				modificationDeleteInterceptorProvider.ProvideReturns(deleteModificationInterceptor)

				deleteModificationInterceptors[entityType] = deleteModificationInterceptor

				resetModificationInterceptors()

				// Register create interceptors
				smb.WithCreateInterceptorProvider(entityType, fakeCreateInterceptorProvider1).Register()
				smb.WithCreateInterceptorProvider(entityType, fakeCreateInterceptorProvider2).
					After(fakeCreateInterceptorProvider1.Name()).Register()

				smb.WithCreateInterceptorProvider(entityType, fakeCreateInterceptorProvider0).
					Before(fakeCreateInterceptorProvider1.Name()).Register()
				smb.WithCreateInterceptorProvider(entityType, fakeCreateInterceptorProviderBA).
					AroundTxBefore(fakeCreateInterceptorProvider0.Name()).
					OnTxAfter(fakeCreateInterceptorProvider2.Name()).
					Register()
				smb.WithCreateInterceptorProvider(entityType, fakeCreateInterceptorProviderAB).
					AroundTxAfter(fakeCreateInterceptorProviderBA.Name()).
					OnTxBefore(fakeCreateInterceptorProviderBA.Name()).
					Register()
				// Register update interceptors
				smb.WithUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProvider1).Register()
				smb.WithUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProvider2).
					After(fakeUpdateInterceptorProvider1.Name()).Register()
				smb.WithUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProvider0).
					Before(fakeUpdateInterceptorProvider1.Name()).Register()
				smb.WithUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProviderBA).
					AroundTxBefore(fakeUpdateInterceptorProvider0.Name()).
					OnTxAfter(fakeUpdateInterceptorProvider2.Name()).
					Register()
				smb.WithUpdateInterceptorProvider(entityType, fakeUpdateInterceptorProviderAB).
					AroundTxAfter(fakeUpdateInterceptorProviderBA.Name()).
					OnTxBefore(fakeUpdateInterceptorProviderBA.Name()).
					Register()
				// Register delete interceptors
				smb.WithDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProvider1).Register()
				smb.WithDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProvider2).
					After(fakeDeleteInterceptorProvider1.Name()).Register()
				smb.WithDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProvider0).
					Before(fakeDeleteInterceptorProvider1.Name()).Register()
				smb.WithDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProviderBA).
					AroundTxBefore(fakeDeleteInterceptorProvider0.Name()).
					OnTxAfter(fakeDeleteInterceptorProvider2.Name()).
					Register()
				smb.WithDeleteInterceptorProvider(entityType, fakeDeleteInterceptorProviderAB).
					AroundTxAfter(fakeDeleteInterceptorProviderBA.Name()).
					OnTxBefore(fakeDeleteInterceptorProviderBA.Name()).
					Register()

				smb.WithCreateInterceptorProvider(entityType, modificationCreateInterceptorProvider).Register()
				smb.WithUpdateInterceptorProvider(entityType, modificationUpdateInterceptorProvider).Register()
				smb.WithDeleteInterceptorProvider(entityType, modificationDeleteInterceptorProvider).Register()

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
	})

	AfterEach(func() {
		resetModificationInterceptors()
		if ctx != nil {
			ctx.CleanupAdditionalResources()
		}
	})

	Describe("Calling other intereceptors", func() {
		Context("when interceptor fails", func() {
			It("should cancel the transaction", func() {
				platform1 := ctx.RegisterPlatformAndActivate(false)

				createModificationInterceptors[types.PlatformType].OnTxCreateStub = func(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
					return func(ctx context.Context, txStorage storage.Repository, newObject types.Object) (types.Object, error) {
						By("calling storage update, should call update interceptor")
						byID := query.ByField(query.EqualsOperator, "id", platform1.ID)
						platformFromDB, err := txStorage.Get(ctx, types.PlatformType, byID)
						if err != nil {
							return nil, err
						}
						_, err = txStorage.Update(ctx, platformFromDB, types.LabelChanges{})
						if err != nil {
							return nil, err
						}
						return f(ctx, txStorage, newObject)
					}
				}

				updateModificationInterceptors[types.PlatformType].OnTxUpdateStub = func(f storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
					return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*types.LabelChange) (object types.Object, e error) {
						return nil, errors.New("expected update to fail")
					}
				}

				platform2 := common.GenerateRandomPlatform()
				ctx.SMWithOAuth.POST(web.PlatformsURL).WithJSON(platform2).Expect().Status(http.StatusInternalServerError)

				ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", web.PlatformsURL, platform2["id"])).
					Expect().Status(http.StatusNotFound)
			})
		})

		Context("when creating platform", func() {
			It("should call all interceptors", func() {
				platform1 := ctx.RegisterPlatformAndActivate(false)
				platform2 := ctx.RegisterPlatformAndActivate(false)

				createModificationInterceptors[types.PlatformType].OnTxCreateStub = func(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
					return func(ctx context.Context, txStorage storage.Repository, newObject types.Object) (types.Object, error) {
						By("calling storage update, should call update interceptor")
						byID := query.ByField(query.EqualsOperator, "id", platform1.ID)
						platformFromDB, err := txStorage.Get(ctx, types.PlatformType, byID)
						if err != nil {
							return nil, err
						}
						_, err = txStorage.Update(ctx, platformFromDB, types.LabelChanges{})
						if err != nil {
							return nil, err
						}
						return f(ctx, txStorage, newObject)
					}
				}

				updateModificationInterceptors[types.PlatformType].OnTxUpdateStub = func(f storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
					return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*types.LabelChange) (object types.Object, e error) {
						deleteCriteria := query.ByField(query.EqualsOperator, "id", platform2.ID)
						By("calling storage delete, should call delete interceptor")
						err := txStorage.Delete(ctx, types.PlatformType, deleteCriteria)
						if err != nil {
							return nil, err
						}
						return f(ctx, txStorage, oldObj, newObj, labelChanges...)
					}
				}

				deleteModificationInterceptors[types.PlatformType].OnTxDeleteStub = func(f storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
					return func(ctx context.Context, txStorage storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) error {
						Expect(len(deletionCriteria)).Should(BeNumerically(">=", 1))
						found := false
						for _, deleteCriteria := range deletionCriteria {
							if deleteCriteria.LeftOp == "id" && deleteCriteria.RightOp[0] == platform2.ID {
								found = true
							}
						}
						if !found {
							Fail("Could not find id criteria")
						}
						return f(ctx, txStorage, objects, deletionCriteria...)
					}
				}

				txDeleteCallCount := deleteModificationInterceptors[types.PlatformType].OnTxDeleteCallCount()
				ctx.RegisterPlatformAndActivate(false)
				Expect(deleteModificationInterceptors[types.PlatformType].OnTxDeleteCallCount()).To(Equal(txDeleteCallCount + 1))

				deleteModificationInterceptors[types.PlatformType].OnTxDeleteStub = func(f storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
					return func(ctx context.Context, txStorage storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) error {
						Expect(len(deletionCriteria)).Should(BeNumerically(">=", 1))
						found := false
						for _, deleteCriteria := range deletionCriteria {
							if deleteCriteria.LeftOp == "id" && deleteCriteria.RightOp[0] == platform1.ID {
								found = true
							}
						}
						if !found {
							Fail("Could not find id criteria")
						}
						return f(ctx, txStorage, objects, deletionCriteria...)
					}
				}

				By("deleting first platform, should call delete interceptor only")
				ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", web.PlatformsURL, platform1.ID)).Expect()
				Expect(deleteModificationInterceptors[types.PlatformType].OnTxDeleteCallCount()).To(Equal(txDeleteCallCount + 2))

				By("should be left with the created platform and the test one only")
				ctx.SMWithOAuth.List(web.PlatformsURL).
					Length().Ge(2)
			})
		})
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
			Expect(deleteStack.Items).To(Equal(getSequence("DeleteReturning", objectType)))
			clearStacks()
		}

		Context("Broker", func() {
			It("Should call interceptors in right order", func() {
				brokerID := ctx.RegisterBrokerWithCatalog(common.NewRandomSBCatalog()).Broker.ID // Post /v1/service_brokers
				checkCreateStack(types.ServiceBrokerType)

				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + brokerID).WithJSON(common.Object{}).Expect().Status(http.StatusOK)
				checkUpdateStack(types.ServiceBrokerType)

				ctx.CleanupBroker(brokerID) // DeleteReturning /v1/service_brokers/<id>
				checkDeleteStack(types.ServiceBrokerType)
			})
		})

		Context("Platform", func() {
			It("Should call interceptors in right order", func() {
				platform := ctx.RegisterPlatformAndActivate(false) // Post /v1/platforms
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
				plans := ctx.SMWithBasic.List(web.ServicePlansURL)
				planID := plans.First().Object().Value("id").String().Raw()
				clearStacks()
				visibility := types.Visibility{
					Base: types.Base{
						Ready: true,
					},
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
					return ctx.RegisterBroker().Broker.ID
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
					plans := ctx.SMWithBasic.List(web.ServicePlansURL)
					planID := plans.First().Object().Value("id").String().Raw()
					visibility := types.Visibility{
						Base: types.Base{
							Ready: true,
						},
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

					createModificationInterceptors[types.ObjectType(e.name)].AroundTxCreateStub = func(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
						return func(ctx context.Context, obj types.Object) (object types.Object, e error) {
							obj.SetLabels(types.Labels{newLabelKey: []string{newLabelValue[0]}})
							obj, e = h(ctx, obj)
							return obj, e
						}
					}
					createModificationInterceptors[types.ObjectType(e.name)].OnTxCreateStub = func(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
						return func(ctx context.Context, txStorage storage.Repository, newObject types.Object) (types.Object, error) {
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
						return func(ctx context.Context, obj types.Object, labelChanges ...*types.LabelChange) (object types.Object, e error) {
							labelChanges = append(labelChanges, &types.LabelChange{
								Key:       newLabelKey,
								Operation: types.AddLabelOperation,
								Values:    []string{newLabelValue[0]},
							})
							return h(ctx, obj, labelChanges...)
						}
					}
					updateModificationInterceptors[types.ObjectType(e.name)].OnTxUpdateStub = func(f storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
						return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*types.LabelChange) (object types.Object, e error) {
							labelChanges = append(labelChanges, &types.LabelChange{
								Key:       newLabelKey,
								Operation: types.AddLabelOperation,
								Values:    []string{newLabelValue[1]},
							})
							return f(ctx, txStorage, oldObj, newObj, labelChanges...)
						}
					}
					entityID := e.createEntryFunc()
					ctx.SMWithOAuth.PATCH(e.url + "/" + entityID).WithJSON(common.Object{}).
						Expect().JSON().Object().
						Value("labels").Object().
						Value(newLabelKey).Array().Equal(newLabelValue)
				})

				It("DeleteReturning", func() {
					_ = e.createEntryFunc()
					deleteModificationInterceptors[types.ObjectType(e.name)].AroundTxDeleteStub = func(h storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
						return func(ctx context.Context, deletionCriteria ...query.Criterion) error {
							deletionCriteria = append(deletionCriteria, query.ByField(query.InOperator, "id", "invalid"))
							return h(ctx, deletionCriteria...)
						}
					}
					ctx.SMWithOAuth.DELETE(e.url).Expect().Status(http.StatusNotFound)
					resetModificationInterceptors()
					deleteModificationInterceptors[types.ObjectType(e.name)].OnTxDeleteStub = func(f storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
						return func(ctx context.Context, txStorage storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) error {
							deletionCriteria = append(deletionCriteria, query.ByField(query.InOperator, "id", "invalid"))
							return f(ctx, txStorage, objects, deletionCriteria...)
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
	fakeCreateInterceptorProvider.NameReturns(name)

	fakeCreateInterceptor := &storagefakes.FakeCreateInterceptor{}
	fakeCreateInterceptor.AroundTxCreateStub = func(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
		return func(ctx context.Context, obj types.Object) (types.Object, error) {
			stack.Add(name + "APIpre")
			obj, err := h(ctx, obj)
			stack.Add(name + "APIpost")
			return obj, err
		}
	}
	fakeCreateInterceptor.OnTxCreateStub = func(h storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
		return func(ctx context.Context, txStorage storage.Repository, newObject types.Object) (types.Object, error) {
			stack.Add(name + "TXpre")
			result, err := h(ctx, txStorage, newObject)
			if err != nil {
				return nil, err
			}
			stack.Add(name + "TXpost")
			return result, nil
		}
	}
	fakeCreateInterceptorProvider.ProvideReturns(fakeCreateInterceptor)
	return fakeCreateInterceptorProvider
}

func updateInterceptorProvider(nameSuffix string, stack *callStack) *storagefakes.FakeUpdateInterceptorProvider {
	name := "Update" + nameSuffix

	fakeUpdateInterceptorProvider := &storagefakes.FakeUpdateInterceptorProvider{}
	fakeUpdateInterceptorProvider.NameReturns(name)
	fakeUpdateInterceptor := &storagefakes.FakeUpdateInterceptor{}
	fakeUpdateInterceptor.AroundTxUpdateStub = func(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
		return func(ctx context.Context, obj types.Object, labelChanges ...*types.LabelChange) (types.Object, error) {
			stack.Add(name + "APIpre")
			obj, err := h(ctx, obj, labelChanges...)
			stack.Add(name + "APIpost")
			return obj, err
		}
	}
	fakeUpdateInterceptor.OnTxUpdateStub = func(h storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
		return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*types.LabelChange) (types.Object, error) {
			stack.Add(name + "TXpre")
			obj, err := h(ctx, txStorage, oldObj, newObj, labelChanges...)
			stack.Add(name + "TXpost")
			return obj, err
		}
	}
	fakeUpdateInterceptorProvider.ProvideReturns(fakeUpdateInterceptor)
	return fakeUpdateInterceptorProvider
}

func deleteInterceptorProvider(nameSuffix string, stack *callStack) *storagefakes.FakeDeleteInterceptorProvider {
	name := "DeleteReturning" + nameSuffix

	fakeDeleteInterceptorProvider := &storagefakes.FakeDeleteInterceptorProvider{}
	fakeDeleteInterceptorProvider.NameReturns(name)
	fakeDeleteInterceptor := &storagefakes.FakeDeleteInterceptor{}
	fakeDeleteInterceptor.AroundTxDeleteStub = func(h storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
		return func(ctx context.Context, deletionCriteria ...query.Criterion) error {
			stack.Add(name + "APIpre")
			err := h(ctx, deletionCriteria...)
			stack.Add(name + "APIpost")
			return err
		}
	}
	fakeDeleteInterceptor.OnTxDeleteStub = func(h storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
		return func(ctx context.Context, txStorage storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) error {
			stack.Add(name + "TXpre")
			err := h(ctx, txStorage, objects, deletionCriteria...)
			stack.Add(name + "TXpost")
			return err
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
