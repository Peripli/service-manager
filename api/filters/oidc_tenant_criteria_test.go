package filters_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/web"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("OIDCTenantCriteriaFilter", func() {
	const (
		clientID           = "client-id"
		labelKey           = "label-key"
		labelValue         = "label-value"
		clientIDTokenClaim = "client-id-claim"
		labelTokenClaim    = "label-token-claim"
	)
	var ctx context.Context
	var clientIDTokenClaimVar string
	var clientIDVar string
	var labelTokenClaimVar string
	var labelValueVar string
	var fakeHandler *webfakes.FakeHandler
	var fakeRequest *web.Request
	var filter *filters.OIDCTenantCriteriaFilter

	BeforeEach(func() {
		ctx = context.TODO()
		fakeHandler = &webfakes.FakeHandler{}
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		Expect(err).ToNot(HaveOccurred())
		fakeRequest = &web.Request{
			Request: req,
		}

		filter = filters.NewOIDCTenantCriteriaFilter(&filters.TenantCriteriaSettings{
			ClientID:           []string{clientID},
			LabelKey:           labelKey,
			ClientIDTokenClaim: clientIDTokenClaim,
			LabelTokenClaim:    labelTokenClaim,
		})

		clientIDTokenClaimVar = clientIDTokenClaim
		clientIDVar = clientID
		labelTokenClaimVar = labelTokenClaim
		labelValueVar = labelValue
		fakeRequest.Request = fakeRequest.Request.WithContext(web.ContextWithUser(ctx, &web.UserContext{
			DataFunc: func(data interface{}) error {
				return json.Unmarshal([]byte(fmt.Sprintf(`{"%s":"%s","%s":"%s"}`, clientIDTokenClaimVar, clientIDVar, labelTokenClaimVar, labelValueVar)), data)
			},
			AuthenticationType: web.Bearer,
			Name:               "test-user",
		}))
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
					DataFunc: func(data interface{}) error {
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
					DataFunc: func(claims interface{}) error {
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
			It("", func() {
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
