package provision_timeout

import (
	. "github.com/onsi/ginkgo"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"net/http"
)

var _ = Describe("Provision timeout", func() {
	Context("when timeout has not reached", func() {
		It("should succeed", func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusOK)

			ctx.SMWithOAuth.List(web.ServiceInstancesURL).Path("$[*].id").Array().Contains(SID)
		}, testTimeout)
	})

	Context("when provision takes more than the configured timeout", func() {
		It("should fail with 500", func() {
			done := make(chan struct{}, 1)
			brokerServer.ServiceInstanceHandler = slowResponseHandler(3, done)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID2).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusInternalServerError)

			ctx.SMWithOAuth.List(web.ServiceInstancesURL).Path("$[*].id").Array().NotContains(SID2)

			verifyOperationDoesNotExist(SID2, "create")
		}, testTimeout)
	})
})
