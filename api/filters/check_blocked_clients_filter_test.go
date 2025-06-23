package filters_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

const (
	TenantLabelKey = "test-tenant"
)

var _ = Describe("CheckBlockedClientFilter", func() {
	var (
		filter      *filters.BlockedClientsFilter
		localCache  *storage.Cache
		ctx         context.Context
		fakeRequest *web.Request
		handler     *webfakes.FakeHandler
		client      types.BlockedClient
		platform    string = `{
  "name": "my-platform",
  "type": "cf",
  "description": "Try our new features.",
  "labels": {
    "%s": [
      "value1"
    ]
  }
}`
	)

	var makeBlockedClient = func() types.BlockedClient {
		blockedClient := types.BlockedClient{}
		id, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		blockedClient.ClientID = id.String()
		blockedClient.SubaccountID = "subb_" + id.String()
		blockedClient.BlockedMethods = []string{"GET", "POST"}
		return blockedClient
	}

	BeforeEach(func() {
		localCache = storage.NewCache(0, nil, nil)
		filter = filters.NewBlockedClientsFilter(localCache, TenantLabelKey)
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		Expect(err).ToNot(HaveOccurred())
		handler = &webfakes.FakeHandler{}
		fakeRequest = &web.Request{
			Request: req,
		}
		client = makeBlockedClient()
		localCache.Add(client.ClientID, client)
	})

	Context("global access", func() {
		It("should not blocked client", func() {
			ctx = web.ContextWithUser(context.Background(), &web.UserContext{
				AuthenticationType: web.Bearer,
				Name:               client.ClientID,
				AccessLevel:        web.GlobalAccess,
			})
			fakeRequest.Request = fakeRequest.WithContext(ctx)
			_, err := filter.Run(fakeRequest, handler)
			Expect(err).ToNot(HaveOccurred())
		})
	})
	Context("tenant access", func() {
		It("should blocked client", func() {
			ctx = web.ContextWithUser(context.Background(), &web.UserContext{
				AuthenticationType: web.Bearer,
				Name:               client.ClientID,
				AccessLevel:        web.TenantAccess,
			})
			fakeRequest.Request = fakeRequest.WithContext(ctx)
			_, err := filter.Run(fakeRequest, handler)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("You're blocked to execute this request. Client"))

		})
	})
	Context("basic access", func() {
		Context("platform is encoded in user context data", func() {
			When("global platform", func() {
				It("should allow to call API", func() {
					ctx = web.ContextWithUser(context.Background(), &web.UserContext{
						AuthenticationType: web.Basic,
						Name:               client.ClientID,
						Data: func(data interface{}) error {
							err := json.Unmarshal([]byte(fmt.Sprintf(platform, TenantLabelKey)), data)
							return err
						}})
					fakeRequest.Request = fakeRequest.WithContext(ctx)
					_, err := filter.Run(fakeRequest, handler)
					Expect(err).To(HaveOccurred())

				})
			})
			When("tenant scoped platform", func() {
				It("should allow to call API", func() {
					ctx = web.ContextWithUser(context.Background(), &web.UserContext{
						AuthenticationType: web.Basic,
						Name:               client.ClientID,
						Data: func(data interface{}) error {
							return json.Unmarshal([]byte(fmt.Sprintf(platform, "global-platform")), data)
						}})
					fakeRequest.Request = fakeRequest.WithContext(ctx)
					_, err := filter.Run(fakeRequest, handler)
					Expect(err).ToNot(HaveOccurred())

				})
			})
		})

		When("platform is not encoded in user context data", func() {
			It("should allow to call API", func() {
				ctx = web.ContextWithUser(context.Background(), &web.UserContext{
					AuthenticationType: web.Basic,
					Name:               client.ClientID,
					Data: func(data interface{}) error {
						return errors.New("new error")
					}})
				fakeRequest.Request = fakeRequest.WithContext(ctx)
				_, err := filter.Run(fakeRequest, handler)
				Expect(err).ToNot(HaveOccurred())

			})
		})

	})
})
