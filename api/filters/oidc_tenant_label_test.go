package filters_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/extensions/table"

	"github.com/benjamintf1/unmarshalledmatchers"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/web"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("OIDCTenantFilters", func() {
	const (
		clientID           = "client-id"
		labelKey           = "label-key"
		labelValue         = "label-value"
		clientIDTokenClaim = "client-id-claim"
		labelTokenClaim    = "label-token-claim"
	)
	var (
		ctx context.Context

		fakeHandler *webfakes.FakeHandler
		fakeRequest *web.Request

		clientIDTokenClaimVar string
		clientIDVar           string
		labelTokenClaimVar    string
		labelValueVar         string

		filter *filters.OIDCTenantFilter
	)

	BeforeEach(func() {
		ctx = context.TODO()
		fakeHandler = &webfakes.FakeHandler{}
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		Expect(err).ToNot(HaveOccurred())
		fakeRequest = &web.Request{
			Request: req,
		}

		clientIDTokenClaimVar = clientIDTokenClaim
		clientIDVar = clientID
		labelTokenClaimVar = labelTokenClaim
		labelValueVar = labelValue
		fakeRequest.Request = fakeRequest.Request.WithContext(web.ContextWithUser(ctx, &web.UserContext{
			Data: func(data interface{}) error {
				return json.Unmarshal([]byte(fmt.Sprintf(`{"%s":"%s","%s":"%s"}`, clientIDTokenClaimVar, clientIDVar, labelTokenClaimVar, labelValueVar)), data)
			},
			AuthenticationType: web.Bearer,
			Name:               "test-user",
		}))
	})

	Describe("OIDCTenantFilter", func() {
		BeforeEach(func() {
			filter = &filters.OIDCTenantFilter{
				ClientID:           clientID,
				ClientIDTokenClaim: clientIDTokenClaim,
				LabelKey:           labelKey,
				LabelTokenClaim:    labelTokenClaim,
				ProcessFunc: func(request *web.Request, labelKey, labelValue string) error {
					return nil
				},
				FilterName: "TestFilter",
				Methods:    []string{http.MethodTrace},
			}
		})

		Describe("Run", func() {
			When("the config is incomplete", func() {
				It("proceeds with next in chain", func() {
					criteriaFilter := filters.NewOIDCTenantCriteriaFilter(&filters.TenantCriteriaSettings{})

					_, err := criteriaFilter.Run(fakeRequest, fakeHandler)

					Expect(err).ToNot(HaveOccurred())
					Expect(fakeHandler.HandleCallCount()).To(Equal(1))
					actualRequest := fakeHandler.HandleArgsForCall(0)
					Expect(query.CriteriaForContext(actualRequest.Context())).To(BeEmpty())
				})
			})
			When("user is missing from context", func() {
				It("proceeds with next in chain", func() {
					fakeRequest.Request = fakeRequest.Request.WithContext(context.TODO())
					Expect(web.UserFromContext(fakeRequest.Context())).To(BeNil())

					_, err := filter.Run(fakeRequest, fakeHandler)

					Expect(err).ToNot(HaveOccurred())
					Expect(fakeHandler.HandleCallCount()).To(Equal(1))
					actualRequest := fakeHandler.HandleArgsForCall(0)
					Expect(query.CriteriaForContext(actualRequest.Context())).To(BeEmpty())
				})
			})

			When("context user is not Bearer token", func() {
				It("proceeds with next in chain", func() {
					fakeRequest.Request = fakeRequest.Request.WithContext(web.ContextWithUser(ctx, &web.UserContext{
						Data: func(data interface{}) error {
							return nil
						},
						AuthenticationType: web.Basic,
						Name:               "test-user",
					}))

					_, err := filter.Run(fakeRequest, fakeHandler)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeHandler.HandleCallCount()).To(Equal(1))
					actualRequest := fakeHandler.HandleArgsForCall(0)
					Expect(query.CriteriaForContext(actualRequest.Context())).To(BeEmpty())
				})
			})

			When("getting claims from token fails", func() {
				It("returns an error", func() {
					fakeRequest.Request = fakeRequest.Request.WithContext(web.ContextWithUser(ctx, &web.UserContext{
						Data: func(claims interface{}) error {
							return fmt.Errorf("error")
						},
						AuthenticationType: web.Bearer,
						Name:               "test-user",
					}))

					_, err := filter.Run(fakeRequest, fakeHandler)
					Expect(err).To(HaveOccurred())

					Expect(fakeHandler.HandleCallCount()).To(Equal(0))
				})
			})

			When("client ID from token claims does not match the filter client ID", func() {
				It("proceeds with next in chain", func() {
					clientIDVar = "different-client-id"

					_, err := filter.Run(fakeRequest, fakeHandler)

					Expect(err).ToNot(HaveOccurred())
					Expect(fakeHandler.HandleCallCount()).To(Equal(1))
					actualRequest := fakeHandler.HandleArgsForCall(0)
					Expect(query.CriteriaForContext(actualRequest.Context())).To(BeEmpty())
				})
			})

			When("client ID token claim is not found in the token claims", func() {
				It("proceeds with next in chain", func() {
					clientIDTokenClaimVar = "different-value"

					_, err := filter.Run(fakeRequest, fakeHandler)

					Expect(err).ToNot(HaveOccurred())
					Expect(fakeHandler.HandleCallCount()).To(Equal(1))
					actualRequest := fakeHandler.HandleArgsForCall(0)
					Expect(query.CriteriaForContext(actualRequest.Context())).To(BeEmpty())
				})
			})

			When("label token claim is not found in the token claims", func() {
				It("returns an error", func() {
					labelTokenClaimVar = "different-value"

					_, err := filter.Run(fakeRequest, fakeHandler)

					Expect(err).To(HaveOccurred())
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

	Describe("OIDCTenantCriteriaFilter", func() {
		BeforeEach(func() {
			filter = filters.NewOIDCTenantCriteriaFilter(&filters.TenantCriteriaSettings{
				ClientID:           clientID,
				LabelKey:           labelKey,
				ClientIDTokenClaim: clientIDTokenClaim,
				LabelTokenClaim:    labelTokenClaim,
			})
		})

		Describe("Run", func() {
			It("invokes the next handler with a request that contains the expected label query", func() {
				expectedCriteria := query.ByLabel(query.EqualsOperator, labelKey, labelValue)

				_, err := filter.Run(fakeRequest, fakeHandler)

				Expect(err).ToNot(HaveOccurred())
				actualRequest := fakeHandler.HandleArgsForCall(0)
				Expect(query.CriteriaForContext(actualRequest.Context())).To(ContainElement(expectedCriteria))
				Expect(fakeHandler.HandleCallCount()).To(Equal(1))
			})
		})
	})

	Describe("OIDCTenantLabelingFilter", func() {
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
					}`, labelKey, labelValue),
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
					}`, labelKey, labelValue),
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
					}`, labelKey, labelValue),
				}),
		}

		DescribeTable("Run", func(t testCase) {
			filter = filters.NewOIDCTenantLabelingFilter(&filters.TenantCriteriaSettings{
				ClientID:           clientID,
				LabelKey:           labelKey,
				ClientIDTokenClaim: clientIDTokenClaim,
				LabelTokenClaim:    labelTokenClaim,
			})

			fakeRequest.Body = []byte(t.actualRequestBody)
			_, err := filter.Run(fakeRequest, fakeHandler)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(string(fakeRequest.Body))).To(unmarshalledmatchers.MatchOrderedJSON(t.expectedRequestBody))
		}, entries...)
	})
})
