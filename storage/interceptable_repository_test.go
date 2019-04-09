package storage_test

import (
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
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
					f := func() {
						interceptableRepository.AddCreateInterceptorProviders(types.ServiceBrokerType, &createProviderMock{name: "Create"})
					}
					f()
					Expect(f).To(Panic())
				})
			})
		})

		Context("Update interceptor", func() {
			Context("When provider with the same name is already registered", func() {
				It("Panics", func() {
					f := func() {
						interceptableRepository.AddUpdateInterceptorProviders(types.ServiceBrokerType, &updateProviderMock{name: "Update"})
					}
					f()
					Expect(f).To(Panic())
				})
			})
		})

		Context("Delete interceptor", func() {
			Context("When provider with the same name is already registered", func() {
				It("Panics", func() {
					f := func() {
						interceptableRepository.AddDeleteInterceptorProviders(types.ServiceBrokerType, &deleteProviderMock{name: "Delete"})
					}
					f()
					Expect(f).To(Panic())
				})
			})
		})
	})
})

type createProviderMock struct {
	name string
}

func (p *createProviderMock) Provide() storage.CreateInterceptor {
	return &createInterceptorMock{p.name}
}

type createInterceptorMock struct {
	name string
}

func (t *createInterceptorMock) Name() string {
	return t.name
}

func (t *createInterceptorMock) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return h
}

func (t *createInterceptorMock) OnTxCreate(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return f
}

type updateProviderMock struct {
	name string
}

func (p *updateProviderMock) Provide() storage.UpdateInterceptor {
	return &updateInterceptorMock{p.name}
}

type updateInterceptorMock struct {
	name string
}

func (u *updateInterceptorMock) Name() string {
	return u.name
}

func (u *updateInterceptorMock) AroundTxUpdate(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return h
}

func (u *updateInterceptorMock) OnTxUpdate(f storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
	return f
}

type deleteProviderMock struct {
	name string
}

func (p *deleteProviderMock) Provide() storage.DeleteInterceptor {
	return &deleteInterceptorMock{p.name}
}

type deleteInterceptorMock struct {
	name string
}

func (u *deleteInterceptorMock) Name() string {
	return u.name
}

func (u *deleteInterceptorMock) AroundTxDelete(h storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
	return h
}

func (u *deleteInterceptorMock) OnTxDelete(f storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
	return f
}
