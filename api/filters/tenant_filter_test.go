package filters_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/benjamintf1/unmarshalledmatchers"

	. "github.com/onsi/ginkgo/extensions/table"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"

	. "github.com/onsi/gomega"

	. "github.com/onsi/ginkgo"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/api/filters"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web/webfakes"
)

var _ = Describe("TenantFilters", func() {
	const (
		labelKey = "label-key"
		tenant   = "tenantID"
	)
	var (
		fakeHandler *webfakes.FakeHandler
		fakeRequest *web.Request
	)

	BeforeEach(func() {
		fakeHandler = &webfakes.FakeHandler{}
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		Expect(err).ToNot(HaveOccurred())
		fakeRequest = &web.Request{
			Request: req,
		}
	})

	Describe("NewMultitenancyFilters", func() {
		var multitenancyFilters []web.Filter

		Describe("Invalid creation parameters", func() {
			When("extractTenantFunc is not provided", func() {
				It("should fail with appropriate message", func() {
					var err error
					multitenancyFilters, err = filters.NewMultitenancyFilters(labelKey, nil)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("extractTenantFunc should be provided"))
				})
			})
		})

		Describe("creation succeeds", func() {
			JustBeforeEach(func() {
				var err error
				multitenancyFilters, err = filters.NewMultitenancyFilters(labelKey, func(request *web.Request) (string, error) {
					return tenant, nil
				})
				Expect(err).ToNot(HaveOccurred())
			})

			Describe("TenantLabelingFilterName", func() {
				It("should return the name of the tenant labeling filter", func() {
					actualFilterName := multitenancyFilters[1].Name()
					Expect(filters.TenantLabelingFilterName()).To(BeEquivalentTo(actualFilterName))
				})
			})

			Describe("Criteria filter", func() {
				for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete, http.MethodPost} {
					When(method+" request is sent with tenant scope", func() {
						It("should modify the request criteria", func() {
							newReq, err := http.NewRequest(method, "http://example.com", nil)
							Expect(err).ShouldNot(HaveOccurred())
							fakeRequest.Request = newReq
							fakeRequest.Request = fakeRequest.WithContext(web.ContextWithUser(context.Background(), &web.UserContext{
								AuthenticationType: web.Bearer,
								Name:               "test",
								AccessLevel:        web.TenantAccess,
							}))
							_, err = multitenancyFilters[0].Run(fakeRequest, fakeHandler)
							Expect(err).ToNot(HaveOccurred())
							Expect(fakeHandler.HandleCallCount()).To(Equal(1))
							actualRequest := fakeHandler.HandleArgsForCall(0)
							criteria := query.CriteriaForContext(actualRequest.Context())
							Expect(criteria).To(HaveLen(1))
							Expect(criteria).To(ContainElement(query.ByLabel(query.EqualsOperator, labelKey, tenant)))
						})
					})

					When(method+" request is sent with global scope", func() {
						It("should not modify the request criteria", func() {
							newReq, err := http.NewRequest(method, "http://example.com", nil)
							Expect(err).ShouldNot(HaveOccurred())
							fakeRequest.Request = newReq
							fakeRequest.Request = fakeRequest.WithContext(web.ContextWithUser(context.Background(), &web.UserContext{
								AuthenticationType: web.Bearer,
								Name:               "test",
								AccessLevel:        web.GlobalAccess,
							}))
							_, err = multitenancyFilters[0].Run(fakeRequest, fakeHandler)
							Expect(err).ToNot(HaveOccurred())
							Expect(fakeHandler.HandleCallCount()).To(Equal(1))
							actualRequest := fakeHandler.HandleArgsForCall(0)
							criteria := query.CriteriaForContext(actualRequest.Context())
							Expect(criteria).To(HaveLen(0))
						})
					})

					When(method+" request is sent without user context", func() {
						It("should not modify the request criteria", func() {
							newReq, err := http.NewRequest(method, "http://example.com", nil)
							Expect(err).ShouldNot(HaveOccurred())
							fakeRequest.Request = newReq
							fakeRequest.Request = fakeRequest.WithContext(web.ContextWithUser(context.Background(), nil))
							_, err = multitenancyFilters[0].Run(fakeRequest, fakeHandler)
							Expect(err).ToNot(HaveOccurred())
							Expect(fakeHandler.HandleCallCount()).To(Equal(1))
							actualRequest := fakeHandler.HandleArgsForCall(0)
							criteria := query.CriteriaForContext(actualRequest.Context())
							Expect(criteria).To(HaveLen(0))
						})
					})
				}
			})

			Describe("Labeling filter", func() {

				Describe("Tenant access", func() {
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
						fakeRequest.Request = fakeRequest.WithContext(web.ContextWithUser(context.Background(), &web.UserContext{
							AuthenticationType: web.Bearer,
							Name:               "test",
							AccessLevel:        web.TenantAccess,
						}))
						fakeRequest.Body = []byte(t.actualRequestBody)
						_, err = multitenancyFilters[1].Run(fakeRequest, fakeHandler)
						Expect(err).ToNot(HaveOccurred())
						Expect(string(fakeRequest.Body)).To(unmarshalledmatchers.MatchOrderedJSON(t.expectedRequestBody))
					}, entries...)
				})

				Describe("Global access", func() {
					type testCase struct {
						actualRequestBody string
					}

					entries := []TableEntry{
						Entry("does not modify request body when there are no labels in the request body",
							testCase{
								actualRequestBody: `{
							   "id":"id"
							}`,
							}),
						Entry("does not modify request body when delimiter label is not part of the request body",
							testCase{
								actualRequestBody: `{
							   "id":"id",
							   "labels":{
								  "another-label":[
									 "another-value"
								  ]
							   }
							}`,
							}),
						Entry("does not modify request body when delimiter label is already part of the request body",
							testCase{
								actualRequestBody: fmt.Sprintf(`{
							   "id":"id",
							   "labels":{
								  "%s":[
									 "another-value"
								  ]
							   }
							}`, labelKey),
							}),
					}

					DescribeTable("Run", func(t testCase) {
						newReq, err := http.NewRequest(http.MethodPost, "http://example.com", nil)
						Expect(err).ShouldNot(HaveOccurred())
						fakeRequest.Request = newReq
						fakeRequest.Request = fakeRequest.WithContext(web.ContextWithUser(context.Background(), &web.UserContext{
							AuthenticationType: web.Bearer,
							Name:               "test",
							AccessLevel:        web.GlobalAccess,
						}))
						fakeRequest.Body = []byte(t.actualRequestBody)
						_, err = multitenancyFilters[1].Run(fakeRequest, fakeHandler)
						Expect(err).ToNot(HaveOccurred())
						Expect(string(fakeRequest.Body)).To(unmarshalledmatchers.MatchOrderedJSON(t.actualRequestBody))
					}, entries...)
				})
			})
		})
	})
})
