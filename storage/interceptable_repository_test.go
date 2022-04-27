package storage_test

import (
	"context"
	"time"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/storagefakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Interceptable TransactionalRepository", func() {
	var interceptableRepository *storage.InterceptableTransactionalRepository
	var ctx context.Context
	var updateTime time.Time

	var fakeCreateAroundTxInterceptorProvider *storagefakes.FakeCreateAroundTxInterceptorProvider
	var fakeCreateAroundTxInterceptor *storagefakes.FakeCreateAroundTxInterceptor

	var fakeCreateOnTxInterceptorProvider *storagefakes.FakeCreateOnTxInterceptorProvider
	var fakeCreateOnTxInterceptor *storagefakes.FakeCreateOnTxInterceptor

	var fakeCreateInterceptorProvider *storagefakes.FakeCreateInterceptorProvider
	var fakeCreateInterceptor *storagefakes.FakeCreateInterceptor

	var fakeUpdateAroundTxInterceptorProvider *storagefakes.FakeUpdateAroundTxInterceptorProvider
	var fakeUpdateAroundTxInterceptor *storagefakes.FakeUpdateAroundTxInterceptor

	var fakeUpdateOnTxInterceptorProvider *storagefakes.FakeUpdateOnTxInterceptorProvider
	var fakeUpdateOnTxIntercetptor *storagefakes.FakeUpdateOnTxInterceptor

	var fakeUpdateInterceptorProvider *storagefakes.FakeUpdateInterceptorProvider
	var fakeUpdateIntercetptor *storagefakes.FakeUpdateInterceptor

	var fakeDeleteAroundTxInterceptorProvider *storagefakes.FakeDeleteAroundTxInterceptorProvider
	var fakeDeleteAroundTxInterceptor *storagefakes.FakeDeleteAroundTxInterceptor

	var fakeDeleteOnTxInterceptorProvider *storagefakes.FakeDeleteOnTxInterceptorProvider
	var fakeDeleteOnTxInterceptor *storagefakes.FakeDeleteOnTxInterceptor

	var fakeDeleteInterceptorProvider *storagefakes.FakeDeleteInterceptorProvider
	var fakeDeleteInterceptor *storagefakes.FakeDeleteInterceptor

	var fakeStorage *storagefakes.FakeStorage

	BeforeEach(func() {
		ctx = context.TODO()
		updateTime = time.Now()

		fakeCreateAroundTxInterceptor = &storagefakes.FakeCreateAroundTxInterceptor{}
		fakeCreateAroundTxInterceptor.AroundTxCreateCalls(func(next storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
			return func(ctx context.Context, obj types.Object) (types.Object, error) {
				return next(ctx, obj)
			}
		})

		fakeCreateOnTxInterceptor = &storagefakes.FakeCreateOnTxInterceptor{}
		fakeCreateOnTxInterceptor.OnTxCreateCalls(func(next storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
			return func(ctx context.Context, txStorage storage.Repository, newObject types.Object) (types.Object, error) {
				return next(ctx, txStorage, newObject)
			}
		})

		fakeCreateInterceptor = &storagefakes.FakeCreateInterceptor{}
		fakeCreateInterceptor.OnTxCreateCalls(func(next storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
			return func(ctx context.Context, txStorage storage.Repository, newObject types.Object) (types.Object, error) {
				return next(ctx, txStorage, newObject)
			}
		})
		fakeCreateInterceptor.AroundTxCreateCalls(func(next storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
			return func(ctx context.Context, obj types.Object) (types.Object, error) {
				return next(ctx, obj)
			}
		})

		fakeUpdateAroundTxInterceptor = &storagefakes.FakeUpdateAroundTxInterceptor{}
		fakeUpdateAroundTxInterceptor.AroundTxUpdateCalls(func(next storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
			return func(ctx context.Context, obj types.Object, labelChanges ...*types.LabelChange) (object types.Object, e error) {
				return next(ctx, obj, labelChanges...)
			}
		})

		fakeUpdateOnTxIntercetptor = &storagefakes.FakeUpdateOnTxInterceptor{}
		fakeUpdateOnTxIntercetptor.OnTxUpdateCalls(func(next storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
			return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*types.LabelChange) (object types.Object, e error) {
				return next(ctx, txStorage, oldObj, newObj, labelChanges...)
			}
		})

		fakeUpdateIntercetptor = &storagefakes.FakeUpdateInterceptor{}
		fakeUpdateIntercetptor.OnTxUpdateCalls(func(next storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
			return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*types.LabelChange) (object types.Object, e error) {
				return next(ctx, txStorage, oldObj, newObj, labelChanges...)
			}
		})

		fakeUpdateIntercetptor.AroundTxUpdateCalls(func(next storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
			return func(ctx context.Context, obj types.Object, labelChanges ...*types.LabelChange) (object types.Object, e error) {
				return next(ctx, obj, labelChanges...)
			}
		})

		fakeDeleteAroundTxInterceptor = &storagefakes.FakeDeleteAroundTxInterceptor{}
		fakeDeleteAroundTxInterceptor.AroundTxDeleteCalls(func(next storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
			return func(ctx context.Context, deletionCriteria ...query.Criterion) error {
				return next(ctx, deletionCriteria...)
			}
		})

		fakeDeleteOnTxInterceptor = &storagefakes.FakeDeleteOnTxInterceptor{}
		fakeDeleteOnTxInterceptor.OnTxDeleteCalls(func(next storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
			return func(ctx context.Context, txStorage storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) error {
				return next(ctx, txStorage, objects, deletionCriteria...)
			}
		})

		fakeDeleteInterceptor = &storagefakes.FakeDeleteInterceptor{}
		fakeDeleteInterceptor.OnTxDeleteCalls(func(next storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
			return func(ctx context.Context, txStorage storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) error {
				return next(ctx, txStorage, objects, deletionCriteria...)
			}
		})
		fakeDeleteInterceptor.AroundTxDeleteCalls(func(next storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
			return func(ctx context.Context, deletionCriteria ...query.Criterion) error {
				return next(ctx, deletionCriteria...)
			}
		})

		fakeCreateAroundTxInterceptorProvider = &storagefakes.FakeCreateAroundTxInterceptorProvider{}
		fakeCreateAroundTxInterceptorProvider.NameReturns("fakeCreateAroundTxInterceptor")
		fakeCreateAroundTxInterceptorProvider.ProvideReturns(fakeCreateAroundTxInterceptor)

		fakeCreateOnTxInterceptorProvider = &storagefakes.FakeCreateOnTxInterceptorProvider{}
		fakeCreateOnTxInterceptorProvider.NameReturns("fakeCreateOnTxInterceptor")
		fakeCreateOnTxInterceptorProvider.ProvideReturns(fakeCreateOnTxInterceptor)

		fakeCreateInterceptorProvider = &storagefakes.FakeCreateInterceptorProvider{}
		fakeCreateInterceptorProvider.NameReturns("fakeCreateInterceptor")
		fakeCreateInterceptorProvider.ProvideReturns(fakeCreateInterceptor)

		fakeUpdateAroundTxInterceptorProvider = &storagefakes.FakeUpdateAroundTxInterceptorProvider{}
		fakeUpdateAroundTxInterceptorProvider.NameReturns("fakeUpdateAroundTxInterceptor")
		fakeUpdateAroundTxInterceptorProvider.ProvideReturns(fakeUpdateAroundTxInterceptor)

		fakeUpdateOnTxInterceptorProvider = &storagefakes.FakeUpdateOnTxInterceptorProvider{}
		fakeUpdateOnTxInterceptorProvider.NameReturns("fakeUpdateOnTxInterceptor")
		fakeUpdateOnTxInterceptorProvider.ProvideReturns(fakeUpdateOnTxIntercetptor)

		fakeUpdateInterceptorProvider = &storagefakes.FakeUpdateInterceptorProvider{}
		fakeUpdateInterceptorProvider.NameReturns("fakeUpdateInterceptor")
		fakeUpdateInterceptorProvider.ProvideReturns(fakeUpdateIntercetptor)

		fakeDeleteAroundTxInterceptorProvider = &storagefakes.FakeDeleteAroundTxInterceptorProvider{}
		fakeDeleteAroundTxInterceptorProvider.NameReturns("fakeDeleteAroundTxInterceptor")
		fakeDeleteAroundTxInterceptorProvider.ProvideReturns(fakeDeleteAroundTxInterceptor)

		fakeDeleteOnTxInterceptorProvider = &storagefakes.FakeDeleteOnTxInterceptorProvider{}
		fakeDeleteOnTxInterceptorProvider.NameReturns("fakeDeleteOnTxInterceptor")
		fakeDeleteOnTxInterceptorProvider.ProvideReturns(fakeDeleteOnTxInterceptor)

		fakeDeleteInterceptorProvider = &storagefakes.FakeDeleteInterceptorProvider{}
		fakeDeleteInterceptorProvider.NameReturns("fakeDeleteInterceptor")
		fakeDeleteInterceptorProvider.ProvideReturns(fakeDeleteInterceptor)

		fakeStorage = &storagefakes.FakeStorage{}
		fakeStorage.InTransactionCalls(func(context context.Context, f func(ctx context.Context, storage storage.Repository) error) error {
			return f(context, fakeStorage)
		})

		fakeStorage.UpdateCalls(func(ctx context.Context, obj types.Object, labelChanges types.LabelChanges, criteria ...query.Criterion) (types.Object, error) {
			return obj, nil
		})

		fakeStorage.GetReturns(&types.ServiceBroker{
			Base: types.Base{
				UpdatedAt: updateTime,
				Ready:     true,
			},
		}, nil)

		fakeStorage.ListNoLabelsReturns(types.NewObjectArray(&types.ServiceBroker{
			Base: types.Base{
				UpdatedAt: updateTime,
				Ready:     true,
			},
		}), nil)

		fakeStorage.GetForUpdateReturns(&types.ServiceBroker{
			Base: types.Base{
				UpdatedAt: updateTime,
				Ready:     true,
			},
		}, nil)

		interceptableRepository = storage.NewInterceptableTransactionalRepository(fakeStorage)

		orderNone := storage.InterceptorOrder{
			OnTxPosition: storage.InterceptorPosition{
				PositionType: storage.PositionNone,
			},
			AroundTxPosition: storage.InterceptorPosition{
				PositionType: storage.PositionNone,
			},
		}

		interceptableRepository.AddCreateAroundTxInterceptorProvider(types.ServiceBrokerType, fakeCreateAroundTxInterceptorProvider, orderNone)
		interceptableRepository.AddCreateOnTxInterceptorProvider(types.ServiceBrokerType, fakeCreateOnTxInterceptorProvider, orderNone)
		interceptableRepository.AddCreateInterceptorProvider(types.ServiceBrokerType, fakeCreateInterceptorProvider, orderNone)

		interceptableRepository.AddUpdateAroundTxInterceptorProvider(types.ServiceBrokerType, fakeUpdateAroundTxInterceptorProvider, orderNone)
		interceptableRepository.AddUpdateOnTxInterceptorProvider(types.ServiceBrokerType, fakeUpdateOnTxInterceptorProvider, orderNone)
		interceptableRepository.AddUpdateInterceptorProvider(types.ServiceBrokerType, fakeUpdateInterceptorProvider, orderNone)

		interceptableRepository.AddDeleteAroundTxInterceptorProvider(types.ServiceBrokerType, fakeDeleteAroundTxInterceptorProvider, orderNone)
		interceptableRepository.AddDeleteOnTxInterceptorProvider(types.ServiceBrokerType, fakeDeleteOnTxInterceptorProvider, orderNone)
		interceptableRepository.AddDeleteInterceptorProvider(types.ServiceBrokerType, fakeDeleteInterceptorProvider, orderNone)

		Expect(fakeCreateAroundTxInterceptor.AroundTxCreateCallCount()).To(Equal(0))
		Expect(fakeCreateOnTxInterceptor.OnTxCreateCallCount()).To(Equal(0))
		Expect(fakeCreateInterceptor.AroundTxCreateCallCount()).To(Equal(0))
		Expect(fakeCreateInterceptor.OnTxCreateCallCount()).To(Equal(0))

		Expect(fakeUpdateAroundTxInterceptor.AroundTxUpdateCallCount()).To(Equal(0))
		Expect(fakeUpdateOnTxIntercetptor.OnTxUpdateCallCount()).To(Equal(0))
		Expect(fakeUpdateIntercetptor.AroundTxUpdateCallCount()).To(Equal(0))
		Expect(fakeUpdateIntercetptor.OnTxUpdateCallCount()).To(Equal(0))

		Expect(fakeDeleteAroundTxInterceptor.AroundTxDeleteCallCount()).To(Equal(0))
		Expect(fakeDeleteOnTxInterceptor.OnTxDeleteCallCount()).To(Equal(0))
		Expect(fakeDeleteInterceptor.AroundTxDeleteCallCount()).To(Equal(0))
		Expect(fakeDeleteInterceptor.OnTxDeleteCallCount()).To(Equal(0))

		Expect(fakeStorage.CreateCallCount()).To(Equal(0))
		Expect(fakeStorage.UpdateCallCount()).To(Equal(0))
		Expect(fakeStorage.DeleteCallCount()).To(Equal(0))
	})

	Describe("Create", func() {
		It("invokes all interceptors", func() {
			_, err := interceptableRepository.Create(ctx, &types.ServiceBroker{})

			Expect(err).ShouldNot(HaveOccurred())

			Expect(fakeCreateAroundTxInterceptor.AroundTxCreateCallCount()).To(Equal(1))
			Expect(fakeCreateOnTxInterceptor.OnTxCreateCallCount()).To(Equal(1))
			Expect(fakeCreateInterceptor.AroundTxCreateCallCount()).To(Equal(1))
			Expect(fakeCreateInterceptor.OnTxCreateCallCount()).To(Equal(1))

			Expect(fakeStorage.CreateCallCount()).To(Equal(1))
		})
	})

	Describe("Update", func() {
		It("invokes all interceptors", func() {
			_, err := interceptableRepository.Update(ctx, &types.ServiceBroker{
				Base: types.Base{
					UpdatedAt: updateTime,
					Ready:     true,
				},
			}, types.LabelChanges{})

			Expect(err).ShouldNot(HaveOccurred())

			Expect(fakeUpdateAroundTxInterceptor.AroundTxUpdateCallCount()).To(Equal(1))
			Expect(fakeUpdateOnTxIntercetptor.OnTxUpdateCallCount()).To(Equal(1))
			Expect(fakeUpdateIntercetptor.AroundTxUpdateCallCount()).To(Equal(1))
			Expect(fakeUpdateIntercetptor.OnTxUpdateCallCount()).To(Equal(1))

			Expect(fakeStorage.UpdateCallCount()).To(Equal(1))
		})
	})

	Describe("UpdateLabels", func() {
		It("invokes all interceptors", func() {
			err := interceptableRepository.UpdateLabels(ctx, types.ServiceBrokerType, "id", types.LabelChanges{})
			Expect(err).ShouldNot(HaveOccurred())

			Expect(fakeUpdateAroundTxInterceptor.AroundTxUpdateCallCount()).To(Equal(1))
			Expect(fakeUpdateOnTxIntercetptor.OnTxUpdateCallCount()).To(Equal(1))
			Expect(fakeUpdateIntercetptor.AroundTxUpdateCallCount()).To(Equal(1))
			Expect(fakeUpdateIntercetptor.OnTxUpdateCallCount()).To(Equal(1))

			Expect(fakeStorage.UpdateLabelsCallCount()).To(Equal(1))
		})
	})

	Describe("Delete", func() {
		It("invokes all interceptors", func() {
			byID := query.ByField(query.EqualsOperator, "id", "id")
			err := interceptableRepository.Delete(ctx, types.ServiceBrokerType, byID)

			Expect(err).ShouldNot(HaveOccurred())

			Expect(fakeDeleteAroundTxInterceptor.AroundTxDeleteCallCount()).To(Equal(1))
			Expect(fakeDeleteOnTxInterceptor.OnTxDeleteCallCount()).To(Equal(1))
			Expect(fakeDeleteInterceptor.AroundTxDeleteCallCount()).To(Equal(1))
			Expect(fakeDeleteInterceptor.OnTxDeleteCallCount()).To(Equal(1))

			Expect(fakeStorage.DeleteCallCount()).To(Equal(1))
		})
	})

	Describe("DeleteReturning", func() {
		It("invokes all interceptors", func() {
			byID := query.ByField(query.EqualsOperator, "id", "id")
			_, err := interceptableRepository.DeleteReturning(ctx, types.ServiceBrokerType, byID)

			Expect(err).ShouldNot(HaveOccurred())

			Expect(fakeDeleteAroundTxInterceptor.AroundTxDeleteCallCount()).To(Equal(1))
			Expect(fakeDeleteOnTxInterceptor.OnTxDeleteCallCount()).To(Equal(1))
			Expect(fakeDeleteInterceptor.AroundTxDeleteCallCount()).To(Equal(1))
			Expect(fakeDeleteInterceptor.OnTxDeleteCallCount()).To(Equal(1))

			Expect(fakeStorage.DeleteReturningCallCount()).To(Equal(1))
		})
	})

	Describe("In transaction", func() {
		var executionsCount int

		OnTxStub := func(ctx context.Context, storage storage.Repository) error {
			for i := 0; i < executionsCount; i++ {
				_, err := storage.Create(ctx, &types.ServiceBroker{})
				Expect(err).ShouldNot(HaveOccurred())

				_, err = storage.Update(ctx, &types.ServiceBroker{
					Base: types.Base{
						UpdatedAt: updateTime,
						Ready:     true,
					},
				}, types.LabelChanges{})
				Expect(err).ShouldNot(HaveOccurred())

				byID := query.ByField(query.EqualsOperator, "id", "id")
				err = storage.Delete(ctx, types.ServiceBrokerType, byID)
				Expect(err).ShouldNot(HaveOccurred())

			}
			return nil
		}

		BeforeEach(func() {
			executionsCount = 1
		})

		Context("when another update happens before the current update has finished", func() {
			BeforeEach(func() {
				fakeStorage.GetCalls(func(ctx context.Context, objectType types.ObjectType, criteria ...query.Criterion) (types.Object, error) {
					return &types.ServiceBroker{
						Base: types.Base{
							// simulate the resource is updated when its retrieved again
							UpdatedAt: updateTime.Add(time.Second),
							Ready:     true,
						},
					}, nil
				})
			})
		})

		Context("when multiple resources of the same type are created/updated/deleted in one transaction", func() {
			BeforeEach(func() {
				executionsCount = 2
			})

			It("triggers the interceptors OnTx of OnTxInterceptors for each resource create/update/delete", func() {
				err := interceptableRepository.InTransaction(ctx, OnTxStub)

				Expect(err).ShouldNot(HaveOccurred())

				Expect(fakeCreateOnTxInterceptor.OnTxCreateCallCount()).To(Equal(executionsCount))
				Expect(fakeUpdateOnTxIntercetptor.OnTxUpdateCallCount()).To(Equal(executionsCount))
				Expect(fakeDeleteOnTxInterceptor.OnTxDeleteCallCount()).To(Equal(executionsCount))

				Expect(fakeStorage.CreateCallCount()).To(Equal(executionsCount))
				Expect(fakeStorage.UpdateCallCount()).To(Equal(executionsCount))
				Expect(fakeStorage.DeleteCallCount()).To(Equal(executionsCount))
			})

			It("does not trigger any aroundTx or mixed interceptors", func() {
				err := interceptableRepository.InTransaction(ctx, OnTxStub)

				Expect(err).ShouldNot(HaveOccurred())

				Expect(fakeCreateInterceptor.AroundTxCreateCallCount()).To(Equal(0))
				Expect(fakeCreateInterceptor.OnTxCreateCallCount()).To(Equal(0))
				Expect(fakeUpdateIntercetptor.AroundTxUpdateCallCount()).To(Equal(0))
				Expect(fakeUpdateIntercetptor.OnTxUpdateCallCount()).To(Equal(0))
				Expect(fakeDeleteInterceptor.AroundTxDeleteCallCount()).To(Equal(0))
				Expect(fakeDeleteInterceptor.OnTxDeleteCallCount()).To(Equal(0))

				Expect(fakeCreateAroundTxInterceptor.AroundTxCreateCallCount()).To(Equal(0))
				Expect(fakeUpdateAroundTxInterceptor.AroundTxUpdateCallCount()).To(Equal(0))
				Expect(fakeDeleteAroundTxInterceptor.AroundTxDeleteCallCount()).To(Equal(0))

				Expect(fakeStorage.CreateCallCount()).To(Equal(executionsCount))
				Expect(fakeStorage.UpdateCallCount()).To(Equal(executionsCount))
				Expect(fakeStorage.DeleteCallCount()).To(Equal(executionsCount))
			})
		})

		It("does not get into infinite recursion when an interceptor triggers the same db op for the same db type it intercepts", func() {
			fakeCreateOnTxInterceptor.OnTxCreateCalls(func(next storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
				return func(ctx context.Context, txStorage storage.Repository, newObject types.Object) (types.Object, error) {
					_, err := txStorage.Create(ctx, newObject)
					Expect(err).ShouldNot(HaveOccurred())

					newObj, err := next(ctx, txStorage, newObject)
					Expect(err).ShouldNot(HaveOccurred())

					_, err = txStorage.Create(ctx, newObject)
					Expect(err).ShouldNot(HaveOccurred())

					return newObj, nil
				}
			})

			fakeUpdateOnTxIntercetptor.OnTxUpdateCalls(func(next storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
				return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*types.LabelChange) (types.Object, error) {
					_, err := txStorage.Update(ctx, newObj, labelChanges)
					Expect(err).ShouldNot(HaveOccurred())

					_, err = next(ctx, txStorage, oldObj, newObj, labelChanges...)
					Expect(err).ShouldNot(HaveOccurred())

					o, err := txStorage.Update(ctx, newObj, labelChanges)
					Expect(err).ShouldNot(HaveOccurred())

					return o, nil
				}
			})

			fakeDeleteOnTxInterceptor.OnTxDeleteCalls(func(next storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
				return func(ctx context.Context, txStorage storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) error {
					byID := query.ByField(query.EqualsOperator, "id", "id")

					err := txStorage.Delete(ctx, types.ServiceBrokerType, byID)
					Expect(err).ShouldNot(HaveOccurred())

					err = next(ctx, txStorage, objects, byID)
					Expect(err).ShouldNot(HaveOccurred())

					err = txStorage.Delete(ctx, types.ServiceBrokerType, byID)
					Expect(err).ShouldNot(HaveOccurred())

					return nil
				}
			})

			err := interceptableRepository.InTransaction(ctx, OnTxStub)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(fakeCreateOnTxInterceptor.OnTxCreateCallCount()).To(Equal(1))
			Expect(fakeUpdateOnTxIntercetptor.OnTxUpdateCallCount()).To(Equal(1))
			Expect(fakeDeleteOnTxInterceptor.OnTxDeleteCallCount()).To(Equal(1))

			Expect(fakeCreateInterceptor.AroundTxCreateCallCount()).To(Equal(0))
			Expect(fakeCreateInterceptor.OnTxCreateCallCount()).To(Equal(0))
			Expect(fakeUpdateIntercetptor.AroundTxUpdateCallCount()).To(Equal(0))
			Expect(fakeUpdateIntercetptor.OnTxUpdateCallCount()).To(Equal(0))
			Expect(fakeDeleteInterceptor.AroundTxDeleteCallCount()).To(Equal(0))
			Expect(fakeDeleteInterceptor.OnTxDeleteCallCount()).To(Equal(0))

			Expect(fakeCreateAroundTxInterceptor.AroundTxCreateCallCount()).To(Equal(0))
			Expect(fakeUpdateAroundTxInterceptor.AroundTxUpdateCallCount()).To(Equal(0))
			Expect(fakeDeleteAroundTxInterceptor.AroundTxDeleteCallCount()).To(Equal(0))

			Expect(fakeStorage.CreateCallCount()).To(Equal(3))
			Expect(fakeStorage.UpdateCallCount()).To(Equal(3))
			Expect(fakeStorage.DeleteCallCount()).To(Equal(3))
		})
	})

	Describe("Register interceptor", func() {
		BeforeEach(func() {
			interceptableRepository = storage.NewInterceptableTransactionalRepository(nil)
		})

		Context("Create interceptor", func() {
			Context("When provider with the same name is already registered", func() {
				It("Panics", func() {
					provider := &storagefakes.FakeCreateInterceptorProvider{}
					provider.NameReturns("createInterceptorProvider")
					f := func() {
						interceptableRepository.AddCreateInterceptorProvider(types.ServiceBrokerType, provider, storage.InterceptorOrder{
							OnTxPosition: storage.InterceptorPosition{
								PositionType: storage.PositionNone,
							},
							AroundTxPosition: storage.InterceptorPosition{
								PositionType: storage.PositionNone,
							},
						})
					}
					f()
					Expect(f).To(Panic())
				})
			})
		})

		Context("Update interceptor", func() {
			Context("When provider with the same name is already registered", func() {
				It("Panics", func() {
					provider := &storagefakes.FakeUpdateInterceptorProvider{}
					provider.NameReturns("updateInterceptorProvider")
					f := func() {
						interceptableRepository.AddUpdateInterceptorProvider(types.ServiceBrokerType, provider, storage.InterceptorOrder{
							OnTxPosition: storage.InterceptorPosition{
								PositionType: storage.PositionNone,
							},
							AroundTxPosition: storage.InterceptorPosition{
								PositionType: storage.PositionNone,
							},
						})
					}
					f()
					Expect(f).To(Panic())
				})
			})
		})

		Context("DeleteReturning interceptor", func() {
			Context("When provider with the same name is already registered", func() {
				It("Panics", func() {
					provider := &storagefakes.FakeDeleteInterceptorProvider{}
					provider.NameReturns("deleteInterceptorProvider")
					f := func() {
						interceptableRepository.AddDeleteInterceptorProvider(types.ServiceBrokerType, provider, storage.InterceptorOrder{
							OnTxPosition: storage.InterceptorPosition{
								PositionType: storage.PositionNone,
							},
							AroundTxPosition: storage.InterceptorPosition{
								PositionType: storage.PositionNone,
							},
						})
					}
					f()
					Expect(f).To(Panic())
				})
			})
		})
	})
})
