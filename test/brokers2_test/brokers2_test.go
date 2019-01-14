package brokers2_test

import (
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/test"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBrokers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker API Tests Suite")
}

var _ = test.DescribeTestsFor(test.TestCase{
	API:            "service_brokers",
	SupportsLabels: false,
	GET: &test.GET{
		ResourceBlueprint: blueprint(true),
	},
	LIST: &test.LIST{
		ResourceBlueprint:                      blueprint(true),
		ResourceWithoutNullableFieldsBlueprint: blueprint(false),
	},
	DELETE: &test.DELETE{
		ResourceCreationBlueprint: blueprint(true),
	},
	DELETELIST: &test.DELETELIST{},
})

func blueprint(withNullableFields bool) func(ctx *common.TestContext) common.Object {
	return func(ctx *common.TestContext) common.Object {
		brokerServer := common.NewBrokerServer()
		UUID, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}
		UUID2, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}
		brokerJSON := common.Object{
			"name":        UUID.String(),
			"broker_url":  brokerServer.URL,
			"description": UUID2.String(),
			"credentials": common.Object{
				"basic": common.Object{
					"username": brokerServer.Username,
					"password": brokerServer.Password,
				},
			},
		}

		if !withNullableFields {
			delete(brokerJSON, "description")
		}
		obj := ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerJSON).
			Expect().
			Status(http.StatusCreated).JSON().Object().Raw()
		delete(obj, "credentials")
		return obj
	}
}
