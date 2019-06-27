package filters_test

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/benjamintf1/unmarshalledmatchers"

	. "github.com/onsi/ginkgo/extensions/table"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/web"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("TenantFilters", func() {
	const (
		labelKey = "label-key"
		tenant   = "tenantID"
	)
	var (
		fakeHandler *webfakes.FakeHandler
		fakeRequest *web.Request

		filter *filters.TenantFilter
	)

	BeforeEach(func() {
		fakeHandler = &webfakes.FakeHandler{}
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		Expect(err).ToNot(HaveOccurred())
		fakeRequest = &web.Request{
			Request: req,
		}
	})

	Describe("TenantFilter", func() {
		var extractTenantFunc func(*web.Request) (string, error)
		var labelingFunc func(request *web.Request, labelKey, labelValue string) error

		BeforeEach(func() {
			extractTenantFunc = func(*web.Request) (string, error) {
				return "", nil
			}
			labelingFunc = func(request *web.Request, labelKey, labelValue string) error {
				return nil
			}
		})

		JustBeforeEach(func() {
			filter = &filters.TenantFilter{
				LabelKey:      labelKey,
				ExtractTenant: extractTenantFunc,
				LabelingFunc:  labelingFunc,
				FilterName:    "TestFilter",
				Methods:       []string{http.MethodTrace},
			}
		})

		Describe("Run", func() {
			When("extract tenant func is not provided", func() {
				BeforeEach(func() {
					extractTenantFunc = nil
				})
				It("should return error", func() {
					_, err := filter.Run(fakeRequest, fakeHandler)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("ExtractTenant"))
				})
			})
			When("labeling func is not provided", func() {
				BeforeEach(func() {
					labelingFunc = nil
				})
				It("should return error", func() {
					_, err := filter.Run(fakeRequest, fakeHandler)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("LabelingFunc"))
				})
			})
			When("the extracted tenant is empty", func() {
				It("proceeds with next in chain", func() {
					_, err := filter.Run(fakeRequest, fakeHandler)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeHandler.HandleCallCount()).To(Equal(1))
					actualRequest := fakeHandler.HandleArgsForCall(0)
					Expect(query.CriteriaForContext(actualRequest.Context())).To(BeEmpty())
				})
			})
			When("extract tenant returns an error", func() {
				BeforeEach(func() {
					extractTenantFunc = func(req *web.Request) (string, error) {
						return "", errors.New("could not extract tenant")
					}
				})
				It("stops the chain", func() {
					_, err := filter.Run(fakeRequest, fakeHandler)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("could not extract tenant"))
					Expect(fakeHandler.HandleCallCount()).To(Equal(0))
				})
			})
			When("labeling function returns an error", func() {
				BeforeEach(func() {
					extractTenantFunc = func(*web.Request) (string, error) {
						return "tenant", nil
					}
					labelingFunc = func(request *web.Request, labelKey string, labelValue string) error {
						return errors.New("could not process label")
					}
				})
				It("stops the chain", func() {
					_, err := filter.Run(fakeRequest, fakeHandler)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("could not process label"))
					Expect(fakeHandler.HandleCallCount()).To(Equal(0))
				})
			})
		})

		Describe("Name", func() {
			It("is not empty", func() {
				Expect(filter.Name()).ToNot(BeEmpty())
			})
		})

		Describe("FilterMatchers", func() {
			It("is not empty", func() {
				Expect(filter.FilterMatchers()).ToNot(BeEmpty())
			})
		})
	})

	Describe("NewMultitenancyFilters", func() {
		var multitenancyFilters []web.Filter
		JustBeforeEach(func() {
			multitenancyFilters = filters.NewMultitenancyFilters(labelKey, func(request *web.Request) (string, error) {
				return tenant, nil
			})
		})

		Describe("Criteria filter", func() {
			for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete} {
				When(method+" request is sent", func() {
					It("should modify the request criteria", func() {
						newReq, err := http.NewRequest(method, "http://example.com", nil)
						Expect(err).ShouldNot(HaveOccurred())
						fakeRequest.Request = newReq
						multitenancyFilters[0].Run(fakeRequest, fakeHandler)
						Expect(fakeHandler.HandleCallCount()).To(Equal(1))
						actualRequest := fakeHandler.HandleArgsForCall(0)
						criteria := query.CriteriaForContext(actualRequest.Context())
						Expect(criteria).To(HaveLen(1))
						Expect(criteria).To(ContainElement(query.ByLabel(query.EqualsOperator, labelKey, tenant)))
					})
				})
			}
		})

		Describe("Labeling filter", func() {
			type testCase struct {
				actualRequestBody   string
				expectedRequestBody string
			}

			entries := []TableEntry{
				Entry("adds the labels array and the specified label key nad value in the request body when there are no labels in the request body",
					testCase{
						actualRequestBody: `{
							   "id":"id"
							}`,
						expectedRequestBody: fmt.Sprintf(`{
						   "id":"id",
						   "labels":{
							  "%s":[
								 "%s"
							  ]
						   }
						}`, labelKey, tenant),
					}),
				Entry("adds the label and the specified value in the request body when delimiter label is not part of the request body",
					testCase{
						actualRequestBody: `{
						   "id":"id",
						   "labels":{
							  "another-label":[
								 "another-value"
							  ]
						   }
						}`,
						expectedRequestBody: fmt.Sprintf(`{
						   "id":"id",
						   "labels":{
							  "another-label":[
								 "another-value"
							  ],
							  "%s":[
								 "%s"
							  ]
						   }
						}`, labelKey, tenant),
					}),
				Entry("appends the specified value to already present label values in the request body when delimiter label is already part of the request body",
					testCase{
						actualRequestBody: fmt.Sprintf(`{
						   "id":"id",
						   "labels":{
							  "%s":[
								 "another-value"
							  ]
						   }
						}`, labelKey),
						expectedRequestBody: fmt.Sprintf(`{
						   "id":"id",
						   "labels":{
							  "%s":[
								 "another-value",
								 "%s"
							  ]
						   }
						}`, labelKey, tenant),
					}),
			}

			DescribeTable("Run", func(t testCase) {
				newReq, err := http.NewRequest(http.MethodPost, "http://example.com", nil)
				Expect(err).ShouldNot(HaveOccurred())
				fakeRequest.Request = newReq
				fakeRequest.Body = []byte(t.actualRequestBody)
				_, err = multitenancyFilters[1].Run(fakeRequest, fakeHandler)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(string(fakeRequest.Body))).To(unmarshalledmatchers.MatchOrderedJSON(t.expectedRequestBody))
			}, entries...)
		})
	})
})
