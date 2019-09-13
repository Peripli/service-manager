package filters_test

import (
	"fmt"
	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("Paging Filter Test", func() {
	var filter web.Filter
	var handler *webfakes.FakeHandler
	var maxItems string
	var request *web.Request

	JustBeforeEach(func() {
		filter = filters.NewPagingFilter(50, 200)
		handler = &webfakes.FakeHandler{}
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://example.com?max_items=%s", maxItems), nil)
		Expect(err).ShouldNot(HaveOccurred())
		request = &web.Request{
			Request: req,
		}
	})

	When("max_items is not a number", func() {
		BeforeEach(func() {
			maxItems = "invalid"
		})
		It("should return error", func() {
			_, err := filter.Run(request, handler)
			Expect(err).Should(HaveOccurred())
			Expect(handler.HandleCallCount()).To(Equal(0))
		})
	})
	When("max_items is negative", func() {
		BeforeEach(func() {
			maxItems = "-1"
		})
		It("should return error", func() {
			_, err := filter.Run(request, handler)
			Expect(err).Should(HaveOccurred())
			Expect(handler.HandleCallCount()).To(Equal(0))
		})
	})
	When("max_items is positive", func() {
		BeforeEach(func() {
			maxItems = "5"
		})
		It("should add limit criteria", func() {
			_, err := filter.Run(request, handler)
			Expect(err).ShouldNot(HaveOccurred())
			actualRequest := handler.HandleArgsForCall(0)
			Expect(query.CriteriaForContext(actualRequest.Context())).To(ContainElement(query.LimitResultBy(5)))
			Expect(actualRequest.Context().Value("limit").(int)).To(Equal(5))
		})
	})
	When("max_items is 0", func() {
		BeforeEach(func() {
			maxItems = "0"
		})
		It("should not add limit criteria", func() {
			_, err := filter.Run(request, handler)
			Expect(err).ShouldNot(HaveOccurred())
			actualRequest := handler.HandleArgsForCall(0)
			criteria := query.CriteriaForContext(actualRequest.Context())
			for _, criterion := range criteria {
				Expect(criterion.LeftOp).ToNot(Equal(query.Limit))
			}
			Expect(actualRequest.Context().Value("limit").(int)).To(Equal(0))
		})
	})

})
