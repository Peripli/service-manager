package storage_test

import (
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/storagefakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Interceptable TransactionalRepository", func() {
	Describe("Register interceptor", func() {
		var interceptableRepository *storage.InterceptableTransactionalRepository

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
