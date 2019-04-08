package storage

import (
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Interceptable TransactionalRepository", func() {
	Describe("Register interceptor", func() {
		var interceptableRepository *InterceptableTransactionalRepository

		BeforeEach(func() {
			interceptableRepository = NewInterceptableTransactionalRepository(nil, nil)
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

func (p *createProviderMock) Provide() CreateInterceptor {
	return &createInterceptorMock{p.name}
}

type createInterceptorMock struct {
	name string
}

func (t *createInterceptorMock) Name() string {
	return t.name
}

func (t *createInterceptorMock) AroundTxCreate(h InterceptCreateAroundTxFunc) InterceptCreateAroundTxFunc {
	return h
}

func (t *createInterceptorMock) OnTxCreate(f InterceptCreateOnTxFunc) InterceptCreateOnTxFunc {
	return f
}

type updateProviderMock struct {
	name string
}

func (p *updateProviderMock) Provide() UpdateInterceptor {
	return &updateInterceptorMock{p.name}
}

type updateInterceptorMock struct {
	name string
}

func (u *updateInterceptorMock) Name() string {
	return u.name
}

func (u *updateInterceptorMock) AroundTxUpdate(h InterceptUpdateAroundTxFunc) InterceptUpdateAroundTxFunc {
	return h
}

func (u *updateInterceptorMock) OnTxUpdate(f InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc {
	return f
}

type deleteProviderMock struct {
	name string
}

func (p *deleteProviderMock) Provide() DeleteInterceptor {
	return &deleteInterceptorMock{p.name}
}

type deleteInterceptorMock struct {
	name string
}

func (u *deleteInterceptorMock) Name() string {
	return u.name
}

func (u *deleteInterceptorMock) AroundTxDelete(h InterceptDeleteAroundTxFunc) InterceptDeleteAroundTxFunc {
	return h
}

func (u *deleteInterceptorMock) OnTxDelete(f InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc {
	return f
}
