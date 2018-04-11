package osb_test

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logic", func() {
	const ()

	var ()

	BeforeEach(func() {
		// build logic
	})

	Describe("GetCatalog", func() {
		assertErrorPropagation := func() {

		}

		assertOSBClientInvoked := func(invocationsCount int) func() {
			return func() {

			}
		}

		callGetCatalog := func(apiVersion string, fail bool) *httptest.ResponseRecorder {
			recorder := httptest.NewRecorder()
			//request, _ := http.NewRequest(http.MethodGet, "/v2/catalog", nil)
			//if apiVersion != "" {
			//	request.Header.Add("X-Broker-API-Version", apiVersion)
			//}
			//request.SetBasicAuth(credentials.Username, credentials.Password)
			//ctx := context.Background()
			//if fail {
			//	ctx = context.WithValue(ctx, "fails", true)
			//}
			//request = request.WithContext(ctx)
			//brokerAPI.ServeHTTP(recorder, request)
			return recorder
		}

		BeforeEach(func() {

		})

		It("invokes the OSB client", assertOSBClientInvoked(1))

		It("returns proper catalog response", func() {

		})

		It("returns no error", func() {
			_, err := logic.GetCatalog(requestContext)

			Expect(err).ToNot(HaveOccurred())
		})

		Context("when an error occurs during OSB client creation", func() {
			BeforeEach(func() {
				// mock to return the error
			})

			It("propagates the error", assertErrorPropagation)

			It("does not invoke the OSB client", assertOSBClientInvoked(0))
		})

		Context("when an error occurs during OSB client call", func() {
			BeforeEach(func() {
				// mock to return the error
			})

			It("invokes the OSB client", assertOSBClientInvoked(1))

			It("propagates the error", assertErrorPropagation)

		})

	})

	Describe("Provision", func() {
		callProvision := func() {

		}

	})

	Describe("Desprovision", func() {

	})

	Describe("Last Operation", func() {

	})

	Describe("Bind", func() {

	})

	Describe("Unbind", func() {

	})

	Describe("Update", func() {

	})

	Describe("Validate Broker API Version", func() {

	})

})
