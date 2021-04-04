package provision_timeout

import (
	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/ginkgo"
	"net/http"
)

var _ = Describe("Provision", func() {
	Context("when provision takes more than the configured timeout", func() {
		It("should fail with 500", func() {
			done := make(chan struct{}, 1)
			brokerServer.ServiceInstanceHandler = slowResponseHandler(3, done)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusInternalServerError)

			ctx.SMWithOAuth.List(web.ServiceInstancesURL).Path("$[*].id").Array().NotContains(SID)

			verifyOperationDoesNotExist(SID, "create")
		}, 10)
	})
})
