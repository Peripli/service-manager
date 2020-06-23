package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/env/envfakes"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/storagefakes"
	"testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestQuery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Query Tests Suite")
}
var store *storage.InterceptableTransactionalRepository;
var _ = Describe("Service Manager Query", func() {
	BeforeSuite(func() {
		store =  storage.NewInterceptableTransactionalRepository(&storagefakes.FakeStorage{})
		fakeEnvironment := &envfakes.FakeEnvironment{}
		fakeEnvironment.AllSettings()
	})


	Context("with 2 notification created at different times", func() {
		BeforeEach(func() {

			storage.NewInterceptableTransactionalRepository(&storagefakes.FakeStorage{})
			mainTest := Maintainer{
				smCtx:                   context.Background(),
				repository:              store,
			}
			mainTest.cleanupExternalOperations()
		})

		It("notifications older than the provided time should not be found", func() {
			// util.ToRFCNanoFormat(now.Add(-time.Hour))
		//	criteria := query.ByField(query.LessThanOperator, "created_at", operand)
		})

	})
})
