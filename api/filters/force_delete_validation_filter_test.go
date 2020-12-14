package filters_test

import (
	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/url"
)

var _ = Describe("Force delete filter test", func() {
	newDeleteRequest := func(path string) *web.Request {
		url, err := url.Parse(path)
		if err != nil {
			Fail(err.Error())
		}
		req := &web.Request{
			Request: &http.Request{
				Method: http.MethodDelete,
				URL:    url,
				Header: http.Header{},
			},
			Body: nil,
		}
		return req
	}

	runFilter := func(path string, force bool, cascade bool) (*web.Response, error) {
		filter := &filters.ForceDeleteValidationFilter{}
		fakeHandler := &webfakes.FakeHandler{}
		fakeHandler.HandleReturns(&web.Response{}, nil)
		request := newDeleteRequest(path)
		if force {
			util.AppendQueryParamToRequest(request, "force", "true")
		}
		if cascade {
			util.AppendQueryParamToRequest(request, "cascade", "true")
		}
		return filter.Run(request, fakeHandler)
	}

	Describe("Run", func() {
		It("should proceed with the request when the URL is allowed", func() {
			allowedPath := web.ServiceInstancesURL
			_, err := runFilter(allowedPath, true, true)
			Expect(err).To(Not(HaveOccurred()))
		})

		It("should fail the request when the URL is not allowed", func() {
			notAllowedPath := web.PlatformsURL
			_, err := runFilter(notAllowedPath, true, true)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(filters.FailOnNotAllowedUrlDescription))
		})

		It("should fail the request when force is true and cascade is false", func() {
			allowedPath := web.ServiceInstancesURL
			_, err := runFilter(allowedPath, true, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(filters.FailOnBadFlagsCombinationDescription))

		})

	})

})
