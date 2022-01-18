package filters_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/api/filters"
	"github.com/benjamintf1/unmarshalledmatchers"
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/web"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("LabelingFilters", func() {
	const (
		labelName = "label-name"
		labelKey  = "label-key"
		value     = "some-value"
	)
	var (
		fakeHandler *webfakes.FakeHandler
		fakeRequest *web.Request

		filter *filters.LabelingFilter
	)

	BeforeEach(func() {
		fakeHandler = &webfakes.FakeHandler{}
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		Expect(err).ToNot(HaveOccurred())
		fakeRequest = &web.Request{
			Request: req,
		}
	})

	Describe("LabelingFilter", func() {
		var ctx context.Context
		var extractValueFunc func(*web.Request) (string, error)
		var labelingFunc func(request *web.Request, labelKey, labelValue string) error

		verifyProceedsWithNextInChain := func() {
			_, err := filter.Run(fakeRequest, fakeHandler)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeHandler.HandleCallCount()).To(Equal(1))
			actualRequest := fakeHandler.HandleArgsForCall(0)
			Expect(query.CriteriaForContext(actualRequest.Context())).To(BeEmpty())
		}

		BeforeEach(func() {
			ctx = web.ContextWithUser(context.Background(), &web.UserContext{
				AuthenticationType: web.Bearer,
				Name:               "test",
				AccessLevel:        web.TenantAccess,
			})
			fakeRequest.Request = fakeRequest.WithContext(ctx)
			extractValueFunc = func(*web.Request) (string, error) {
				return "", nil
			}
			labelingFunc = func(request *web.Request, labelKey, labelValue string) error {
				return nil
			}
		})

		JustBeforeEach(func() {
			filter = &filters.LabelingFilter{
				LabelKey:     labelKey,
				ExtractValue: extractValueFunc,
				LabelingFunc: labelingFunc,
				FilterName:   "TestFilter",
				Methods:      []string{http.MethodTrace},
				BasePaths:    []string{web.ServiceInstancesURL},
			}
		})

		Describe("Run", func() {
			When("extract value func is not provided", func() {
				BeforeEach(func() {
					extractValueFunc = nil
				})
				It("should return error", func() {
					_, err := filter.Run(fakeRequest, fakeHandler)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("ExtractValue"))
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
			When("usercontext is missing", func() {
				BeforeEach(func() {
					fakeRequest.Request = fakeRequest.WithContext(context.TODO())
				})
				It("proceeds with next in chain", verifyProceedsWithNextInChain)
			})
			When("authentication type is not Bearer", func() {
				BeforeEach(func() {
					fakeRequest.Request = fakeRequest.WithContext(web.ContextWithUser(context.Background(), &web.UserContext{
						AuthenticationType: web.Basic,
						AccessLevel:        web.GlobalAccess,
					}))
				})
				It("proceeds with next in chain", verifyProceedsWithNextInChain)
			})
			When("access level is global", func() {
				BeforeEach(func() {
					fakeRequest.Request = fakeRequest.WithContext(web.ContextWithUser(context.Background(), &web.UserContext{
						AuthenticationType: web.Bearer,
						AccessLevel:        web.GlobalAccess,
					}))
				})
				It("proceeds with next in chain", verifyProceedsWithNextInChain)
			})
			When("the extracted value is empty", func() {
				It("proceeds with next in chain", verifyProceedsWithNextInChain)
			})
			When("extract value returns an error", func() {
				BeforeEach(func() {
					extractValueFunc = func(req *web.Request) (string, error) {
						return "", errors.New("could not extract value")
					}
				})
				It("stops the chain", func() {
					_, err := filter.Run(fakeRequest, fakeHandler)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("could not extract value"))
					Expect(fakeHandler.HandleCallCount()).To(Equal(0))
				})
			})
			When("labeling function returns an error", func() {
				BeforeEach(func() {
					extractValueFunc = func(*web.Request) (string, error) {
						return "value", nil
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
			It("has same number of entries as BasePaths", func() {
				Expect(len(filter.FilterMatchers())).To(BeEquivalentTo(len(filter.BasePaths)))
			})
		})
	})

	Describe("LabelingFilters", func() {
		var labelingFilters []web.Filter
		JustBeforeEach(func() {
			labelingFilters = filters.NewLabelingFilters(labelName, labelKey, *new([]string), func(request *web.Request) (string, error) {
				return value, nil
			})
		})

		Describe("Criteria filter", func() {
			for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete} {
				When(method+" request is sent", func() {
					It("should modify the request criteria", func() {
						newReq, err := http.NewRequest(method, "http://example.com", nil)
						Expect(err).ShouldNot(HaveOccurred())
						fakeRequest.Request = newReq
						fakeRequest.Request = fakeRequest.WithContext(web.ContextWithUser(context.Background(), &web.UserContext{
							AuthenticationType: web.Bearer,
							Name:               "test",
							AccessLevel:        web.TenantAccess,
						}))
						_, err = labelingFilters[0].Run(fakeRequest, fakeHandler)
						Expect(err).ToNot(HaveOccurred())
						Expect(fakeHandler.HandleCallCount()).To(Equal(1))
						actualRequest := fakeHandler.HandleArgsForCall(0)
						criteria := query.CriteriaForContext(actualRequest.Context())
						Expect(criteria).To(HaveLen(1))
						Expect(criteria).To(ContainElement(query.ByLabel(query.EqualsOperator, labelKey, value)))
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
						}`, labelKey, value),
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
						}`, labelKey, value),
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
						}`, labelKey, value),
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
				_, err = labelingFilters[1].Run(fakeRequest, fakeHandler)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(fakeRequest.Body)).To(unmarshalledmatchers.MatchOrderedJSON(t.expectedRequestBody))
			}, entries)
		})
	})
})
