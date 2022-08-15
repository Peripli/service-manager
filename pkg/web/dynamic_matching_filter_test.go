package web_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web/webfakes"
)

var _ = Describe("Dynamic filter", func() {
	entries := []TableEntry{
		Entry("with no inner filters", test{
			filters: []*innerTestFilter{},
			called:  []bool{},
			method:  http.MethodGet,
			url:     "http://test.com/test",
		}),
		Entry("with one inner filter", test{
			filters: []*innerTestFilter{
				filterWithMatchers(web.Methods(http.MethodGet)),
			},
			called: []bool{true},
			method: http.MethodGet,
			url:    "http://test.com/test",
		}),
		Entry("with two inner filters for different methods", test{
			filters: []*innerTestFilter{
				filterWithMatchers(web.Methods(http.MethodGet)),
				filterWithMatchers(web.Methods(http.MethodPost)),
			},
			called: []bool{true, false},
			method: http.MethodGet,
			url:    "http://test.com/test",
		}),
		Entry("with two inner filters for different paths", test{
			filters: []*innerTestFilter{
				filterWithMatchers(web.Path("/test2")),
				filterWithMatchers(web.Methods("/test3")),
			},
			called: []bool{false, false},
			method: http.MethodGet,
			url:    "http://test.com/test",
		}),
		Entry("with two inner filters for different paths, one matching", test{
			filters: []*innerTestFilter{
				filterWithMatchers(web.Path("/*")),
				filterWithMatchers(web.Methods("/test3")),
			},
			called: []bool{true, false},
			method: http.MethodGet,
			url:    "http://test.com/test",
		}),
	}

	DescribeTable("Run", func(e test) {
		dynamicFilter := web.NewDynamicMatchingFilter("test")
		fakeHandler := &webfakes.FakeHandler{}
		req, err := http.NewRequest(e.method, e.url, nil)
		Expect(err).ToNot(HaveOccurred())
		fakeRequest := &web.Request{
			Request: req,
		}

		for _, f := range e.filters {
			dynamicFilter.AddFilter(f)
		}

		_, err = dynamicFilter.Run(fakeRequest, fakeHandler)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(fakeHandler.HandleCallCount()).To(Equal(1))
		for i, f := range e.filters {
			Expect(f.called).To(Equal(e.called[i]))
		}

	}, entries...)
})

type test struct {
	filters     []*innerTestFilter
	called      []bool
	url, method string
}

func filterWithMatchers(matchers ...web.Matcher) *innerTestFilter {
	return &innerTestFilter{
		matchers: []web.FilterMatcher{
			{
				Matchers: matchers,
			},
		},
	}
}

type innerTestFilter struct {
	called   bool
	matchers []web.FilterMatcher
}

func (tf *innerTestFilter) Name() string {
	return ""
}

func (tf *innerTestFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	tf.called = true
	return next.Handle(req)
}

func (tf *innerTestFilter) FilterMatchers() []web.FilterMatcher {
	return tf.matchers
}
