package osb

import (
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage/storagefakes"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller", func() {

	var (
		fakeBroker *storagefakes.FakeBroker
		controller *Controller
	)

	Describe("Routes", func() {
		BeforeEach(func() {
			fakeBroker = &storagefakes.FakeBroker{}
			controller = &Controller{
				BrokerStorage: fakeBroker,
			}

		})

		It("returns one valid OSB endpoint with router as handler", func() {
			routes := controller.Routes()
			Expect(len(routes)).To(Equal(1))

			route := routes[0]
			Expect(route.Handler).To(BeAssignableToTypeOf(mux.NewRouter()))
			Expect(route.Endpoint.Path).To(ContainSubstring("/osb"))
			Expect(route.Endpoint.Method).To(Equal(rest.AllMethods))
		})
	})
})
