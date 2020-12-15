package filters_test

import (
	"fmt"
	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/url"
)

var AllowedPaths = []string{web.ServiceInstancesURL, web.ServiceBindingsURL}
var NotAllowedPaths = []string{web.PlatformsURL, web.VisibilitiesURL, web.ServicePlansURL, web.ServiceOfferingsURL, web.ServiceBrokersURL, web.OSBURL}

var _ = Describe("Force delete filter test", func() {
	newDeleteRequest := func(path string) *web.Request {
		url, err := url.Parse("/some-before/" + path + "/some-after")
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
		for _, allowedPath := range AllowedPaths {
			allowedPath := allowedPath
			It(fmt.Sprintf("should proceed with the request when the URL is allowed: %s", allowedPath), func() {
				_, err := runFilter(allowedPath, true, true)
				Expect(err).To(Not(HaveOccurred()))
			})
		}

		for _, notAllowedPath := range NotAllowedPaths {
			notAllowedPath := notAllowedPath
			It(fmt.Sprintf("should fail the request when the URL is not allowed: %s", notAllowedPath), func() {
				_, err := runFilter(notAllowedPath, true, true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(filters.FailOnNotAllowedUrlDescription))
			})
		}

		It("should fail the request when force is true and cascade is false", func() {
			allowedPath := web.ServiceInstancesURL
			_, err := runFilter(allowedPath, true, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(filters.FailOnBadFlagsCombinationDescription))

		})

	})

})
