package storage_test

import (
	"context"

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
		var fakeCreateInterceptorProvider *storagefakes.FakeCreateInterceptorProvider
		var fakeCreateInterceptor *storagefakes.FakeCreateInterceptor

		var fakeUpdateInterceptorProvider *storagefakes.FakeUpdateInterceptorProvider
		var fakeUpdateIntercetptor *storagefakes.FakeUpdateInterceptor

		var fakeDeleteInterceptorProvider *storagefakes.FakeDeleteInterceptorProvider
		var fakeDeleteInterceptor *storagefakes.FakeDeleteInterceptor

		var fakeEncrypter *securityfakes.FakeEncrypter

		var fakeStorage *storagefakes.FakeStorage

		BeforeEach(func() {
			ctx = context.TODO()

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
				return func(ctx context.Context, txStorage storage.Repository, obj types.Object, labelChanges ...*query.LabelChange) (object types.Object, e error) {
					return next(ctx, txStorage, obj, labelChanges...)
				}
			})

			fakeUpdateIntercetptor.AroundTxUpdateCalls(func(next storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
				return func(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (object types.Object, e error) {
					return next(ctx, obj, labelChanges...)
				}
			})

			fakeDeleteInterceptor = &storagefakes.FakeDeleteInterceptor{}
			fakeDeleteInterceptor.OnTxDeleteCalls(func(next storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
				return func(ctx context.Context, txStorage storage.Repository, deletionCriteria ...query.Criterion) (list types.ObjectList, e error) {
					return next(ctx, txStorage, deletionCriteria...)
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

		})

		It("triggers the interceptors OnTx logic", func() {
			Expect(fakeCreateInterceptor.OnTxCreateCallCount()).To(Equal(0))
			Expect(fakeUpdateIntercetptor.OnTxUpdateCallCount()).To(Equal(0))
			Expect(fakeDeleteInterceptor.OnTxDeleteCallCount()).To(Equal(0))

			Expect(fakeCreateInterceptor.AroundTxCreateCallCount()).To(Equal(0))
			Expect(fakeUpdateIntercetptor.AroundTxUpdateCallCount()).To(Equal(0))
			Expect(fakeDeleteInterceptor.AroundTxDeleteCallCount()).To(Equal(0))

			Expect(fakeStorage.CreateCallCount()).To(Equal(0))
			Expect(fakeStorage.UpdateCallCount()).To(Equal(0))
			Expect(fakeStorage.DeleteCallCount()).To(Equal(0))

			err := interceptableRepository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
				_, err := storage.Create(ctx, &types.ServiceBroker{})
				Expect(err).ShouldNot(HaveOccurred())

				_, err = storage.Update(ctx, &types.ServiceBroker{})
				Expect(err).ShouldNot(HaveOccurred())

				byID := query.ByField(query.EqualsOperator, "id", "id")
				_, err = storage.Delete(ctx, types.ServiceBrokerType, byID)
				Expect(err).ShouldNot(HaveOccurred())

				return nil
			})

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
