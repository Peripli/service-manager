package storage_test

import (
	"context"
	"time"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/security/securityfakes"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/storagefakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Interceptable TransactionalRepository", func() {
	var interceptableRepository *storage.InterceptableTransactionalRepository
	var ctx context.Context

	Describe("In transaction", func() {
		var updateTime time.Time
		var fakeCreateInterceptorProvider *storagefakes.FakeCreateInterceptorProvider
		var fakeCreateInterceptor *storagefakes.FakeCreateInterceptor

		var fakeUpdateInterceptorProvider *storagefakes.FakeUpdateInterceptorProvider
		var fakeUpdateIntercetptor *storagefakes.FakeUpdateInterceptor

		var fakeDeleteInterceptorProvider *storagefakes.FakeDeleteInterceptorProvider
		var fakeDeleteInterceptor *storagefakes.FakeDeleteInterceptor

		var fakeEncrypter *securityfakes.FakeEncrypter

		var fakeStorage *storagefakes.FakeStorage

		OnTxStub := func(ctx context.Context, storage storage.Repository) error {
			_, err := storage.Create(ctx, &types.ServiceBroker{})
			Expect(err).ShouldNot(HaveOccurred())

			_, err = storage.Update(ctx, &types.ServiceBroker{
				Base: types.Base{
					UpdatedAt: updateTime,
				},
			})
			Expect(err).ShouldNot(HaveOccurred())

			byID := query.ByField(query.EqualsOperator, "id", "id")
			_, err = storage.Delete(ctx, types.ServiceBrokerType, byID)
			Expect(err).ShouldNot(HaveOccurred())

			return nil
		}

		BeforeEach(func() {
			ctx = context.TODO()
			updateTime = time.Now()

			fakeCreateInterceptor = &storagefakes.FakeCreateInterceptor{}
			fakeCreateInterceptor.OnTxCreateCalls(func(next storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
				return func(ctx context.Context, txStorage storage.Repository, newObject types.Object) error {
					return next(ctx, txStorage, newObject)
				}
			})
			fakeCreateInterceptor.AroundTxCreateCalls(func(next storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
				return func(ctx context.Context, obj types.Object) (types.Object, error) {
					return next(ctx, obj)
				}
			})

			fakeUpdateIntercetptor = &storagefakes.FakeUpdateInterceptor{}
			fakeUpdateIntercetptor.OnTxUpdateCalls(func(next storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
				return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*query.LabelChange) (object types.Object, e error) {
					return next(ctx, txStorage, oldObj, newObj, labelChanges...)
				}
			})

			fakeUpdateIntercetptor.AroundTxUpdateCalls(func(next storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
				return func(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (object types.Object, e error) {
					return next(ctx, obj, labelChanges...)
				}
			})

			fakeDeleteInterceptor = &storagefakes.FakeDeleteInterceptor{}
			fakeDeleteInterceptor.OnTxDeleteCalls(func(next storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
				return func(ctx context.Context, txStorage storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) (list types.ObjectList, e error) {
					return next(ctx, txStorage, objects, deletionCriteria...)
				}
			})
			fakeDeleteInterceptor.AroundTxDeleteCalls(func(next storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
				return func(ctx context.Context, deletionCriteria ...query.Criterion) (list types.ObjectList, e error) {
					return next(ctx, deletionCriteria...)
				}
			})

			fakeCreateInterceptorProvider = &storagefakes.FakeCreateInterceptorProvider{}
			fakeCreateInterceptorProvider.NameReturns("fakeCreateInterceptor")
			fakeCreateInterceptorProvider.ProvideReturns(fakeCreateInterceptor)

			fakeUpdateInterceptorProvider = &storagefakes.FakeUpdateInterceptorProvider{}
			fakeUpdateInterceptorProvider.NameReturns("fakeUpdateInterceptor")
			fakeUpdateInterceptorProvider.ProvideReturns(fakeUpdateIntercetptor)

			fakeDeleteInterceptorProvider = &storagefakes.FakeDeleteInterceptorProvider{}
			fakeDeleteInterceptorProvider.NameReturns("fakeDeleteInterceptor")
			fakeDeleteInterceptorProvider.ProvideReturns(fakeDeleteInterceptor)

			fakeEncrypter = &securityfakes.FakeEncrypter{}

			fakeStorage = &storagefakes.FakeStorage{}
			fakeStorage.InTransactionCalls(func(context context.Context, f func(ctx context.Context, storage storage.Repository) error) error {
				return f(context, fakeStorage)
			})

			fakeStorage.UpdateCalls(func(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
				return obj, nil
			})

			fakeStorage.GetReturns(&types.ServiceBroker{
				Base: types.Base{
					UpdatedAt: updateTime,
				},
			}, nil)

			interceptableRepository = storage.NewInterceptableTransactionalRepository(fakeStorage, fakeEncrypter)

			orderNone := storage.InterceptorOrder{
				OnTxPosition: storage.InterceptorPosition{
					PositionType: storage.PositionNone,
				},
				AroundTxPosition: storage.InterceptorPosition{
					PositionType: storage.PositionNone,
				},
			}

			interceptableRepository.AddCreateInterceptorProvider(types.ServiceBrokerType, storage.OrderedCreateInterceptorProvider{
				InterceptorOrder:          orderNone,
				CreateInterceptorProvider: fakeCreateInterceptorProvider,
			})

			interceptableRepository.AddUpdateInterceptorProvider(types.ServiceBrokerType, storage.OrderedUpdateInterceptorProvider{
				InterceptorOrder:          orderNone,
				UpdateInterceptorProvider: fakeUpdateInterceptorProvider,
			})

			interceptableRepository.AddDeleteInterceptorProvider(types.ServiceBrokerType, storage.OrderedDeleteInterceptorProvider{
				InterceptorOrder:          orderNone,
				DeleteInterceptorProvider: fakeDeleteInterceptorProvider,
			})

			Expect(fakeCreateInterceptor.OnTxCreateCallCount()).To(Equal(0))
			Expect(fakeUpdateIntercetptor.OnTxUpdateCallCount()).To(Equal(0))
			Expect(fakeDeleteInterceptor.OnTxDeleteCallCount()).To(Equal(0))

			Expect(fakeCreateInterceptor.AroundTxCreateCallCount()).To(Equal(0))
			Expect(fakeUpdateIntercetptor.AroundTxUpdateCallCount()).To(Equal(0))
			Expect(fakeDeleteInterceptor.AroundTxDeleteCallCount()).To(Equal(0))

			Expect(fakeStorage.CreateCallCount()).To(Equal(0))
			Expect(fakeStorage.UpdateCallCount()).To(Equal(0))
			Expect(fakeStorage.DeleteCallCount()).To(Equal(0))
		})

		Context("when another update happens before the current update has finished", func() {
			BeforeEach(func() {
				fakeStorage.GetCalls(func(ctx context.Context, objectType types.ObjectType, id string) (types.Object, error) {
					return &types.ServiceBroker{
						Base: types.Base{
							// simulate the resource is updated when its retrieved again
							UpdatedAt: updateTime.Add(time.Second),
						},
					}, nil
				})
			})

			It("fails with concurrent modification failure", func() {
				err := interceptableRepository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {

					_, err := storage.Update(ctx, &types.ServiceBroker{
						Base: types.Base{
							UpdatedAt: updateTime,
						},
					})

					return err
				})

				Expect(err).Should(HaveOccurred())
				Expect(err).To(Equal(util.ErrConcurrentResourceModification))
			})
		})

		It("triggers the interceptors OnTx logic", func() {
			err := interceptableRepository.InTransaction(ctx, OnTxStub)

			Expect(err).ShouldNot(HaveOccurred())

			Expect(fakeCreateInterceptor.OnTxCreateCallCount()).To(Equal(1))
			Expect(fakeUpdateIntercetptor.OnTxUpdateCallCount()).To(Equal(1))
			Expect(fakeDeleteInterceptor.OnTxDeleteCallCount()).To(Equal(1))

			Expect(fakeCreateInterceptor.AroundTxCreateCallCount()).To(Equal(0))
			Expect(fakeUpdateIntercetptor.AroundTxUpdateCallCount()).To(Equal(0))
			Expect(fakeDeleteInterceptor.AroundTxDeleteCallCount()).To(Equal(0))

			Expect(fakeStorage.CreateCallCount()).To(Equal(1))
			Expect(fakeStorage.UpdateCallCount()).To(Equal(1))
			Expect(fakeStorage.DeleteCallCount()).To(Equal(1))
		})

		It("does not get into infinite recursion when an interceptor triggers the same db op for the same db type it intercepts", func() {
			fakeCreateInterceptor.OnTxCreateCalls(func(next storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
				return func(ctx context.Context, txStorage storage.Repository, newObject types.Object) error {
					_, err := txStorage.Create(ctx, newObject)
					Expect(err).ShouldNot(HaveOccurred())

					err = next(ctx, txStorage, newObject)
					Expect(err).ShouldNot(HaveOccurred())

					_, err = txStorage.Create(ctx, newObject)
					Expect(err).ShouldNot(HaveOccurred())

					return nil
				}
			})

			fakeUpdateIntercetptor.OnTxUpdateCalls(func(next storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
				return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
					o, err := txStorage.Update(ctx, newObj, labelChanges...)
					Expect(err).ShouldNot(HaveOccurred())

					o, err = next(ctx, txStorage, oldObj, newObj, labelChanges...)
					Expect(err).ShouldNot(HaveOccurred())

					o, err = txStorage.Update(ctx, newObj, labelChanges...)
					Expect(err).ShouldNot(HaveOccurred())

					return o, nil
				}
			})

			fakeDeleteInterceptor.OnTxDeleteCalls(func(next storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
				return func(ctx context.Context, txStorage storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
					byID := query.ByField(query.EqualsOperator, "id", "id")

					objectList, err := txStorage.Delete(ctx, types.ServiceBrokerType, byID)
					Expect(err).ShouldNot(HaveOccurred())

					objectList, err = next(ctx, txStorage, objects, byID)
					Expect(err).ShouldNot(HaveOccurred())

					objectList, err = txStorage.Delete(ctx, types.ServiceBrokerType, byID)
					Expect(err).ShouldNot(HaveOccurred())

					return objectList, nil
				}
			})

			err := interceptableRepository.InTransaction(ctx, OnTxStub)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(fakeCreateInterceptor.OnTxCreateCallCount()).To(Equal(1))
			Expect(fakeUpdateIntercetptor.OnTxUpdateCallCount()).To(Equal(1))
			Expect(fakeDeleteInterceptor.OnTxDeleteCallCount()).To(Equal(1))

			Expect(fakeCreateInterceptor.AroundTxCreateCallCount()).To(Equal(0))
			Expect(fakeUpdateIntercetptor.AroundTxUpdateCallCount()).To(Equal(0))
			Expect(fakeDeleteInterceptor.AroundTxDeleteCallCount()).To(Equal(0))

			Expect(fakeStorage.CreateCallCount()).To(Equal(3))
			Expect(fakeStorage.UpdateCallCount()).To(Equal(3))
			Expect(fakeStorage.DeleteCallCount()).To(Equal(3))

		})
	})

	Describe("Register interceptor", func() {

		BeforeEach(func() {
			interceptableRepository = storage.NewInterceptableTransactionalRepository(nil, nil)
		})

		Context("Create interceptor", func() {
			Context("When provider with the same name is already registered", func() {
				It("Panics", func() {
					provider := &storagefakes.FakeCreateInterceptorProvider{}
					provider.NameReturns("createInterceptorProvider")
					f := func() {
						interceptableRepository.AddCreateInterceptorProvider(types.ServiceBrokerType, storage.OrderedCreateInterceptorProvider{
							InterceptorOrder: storage.InterceptorOrder{
								OnTxPosition: storage.InterceptorPosition{
									PositionType: storage.PositionNone,
								},
								AroundTxPosition: storage.InterceptorPosition{
									PositionType: storage.PositionNone,
								},
							},
							CreateInterceptorProvider: provider,
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
						interceptableRepository.AddUpdateInterceptorProvider(types.ServiceBrokerType, storage.OrderedUpdateInterceptorProvider{
							InterceptorOrder: storage.InterceptorOrder{
								OnTxPosition: storage.InterceptorPosition{
									PositionType: storage.PositionNone,
								},
								AroundTxPosition: storage.InterceptorPosition{
									PositionType: storage.PositionNone,
								},
							},
							UpdateInterceptorProvider: provider,
						})
					}
					f()
					Expect(f).To(Panic())
				})
			})
		})

		Context("Delete interceptor", func() {
			Context("When provider with the same name is already registered", func() {
				It("Panics", func() {
					provider := &storagefakes.FakeDeleteInterceptorProvider{}
					provider.NameReturns("deleteInterceptorProvider")
					f := func() {
						interceptableRepository.AddDeleteInterceptorProvider(types.ServiceBrokerType, storage.OrderedDeleteInterceptorProvider{
							InterceptorOrder: storage.InterceptorOrder{
								OnTxPosition: storage.InterceptorPosition{
									PositionType: storage.PositionNone,
								},
								AroundTxPosition: storage.InterceptorPosition{
									PositionType: storage.PositionNone,
								},
							},
							DeleteInterceptorProvider: provider,
						})
					}
					f()
					Expect(f).To(Panic())
				})
			})
		})
	})
})
