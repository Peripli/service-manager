package platforms2_test

import (
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test"
)

func TestPlatforms(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Platforms Tests Suite")

}

var _ = test.DescribeTestsFor(test.TestCase{
	API:            "platforms",
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
	DELETELIST: &test.DELETELIST{
		ResourceBlueprint:                      blueprint(true),
		ResourceWithoutNullableFieldsBlueprint: blueprint(false),
	},
},
)

func blueprint(withNullableFields bool) func(ctx *common.TestContext) common.Object {
	return func(ctx *common.TestContext) common.Object {
		randomPlatform := common.GenerateRandomPlatform()
		if !withNullableFields {
			delete(randomPlatform, "description")
		}
		platform := ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(randomPlatform).
			Expect().
			Status(http.StatusCreated).JSON().Object().Raw()
		delete(platform, "credentials")

		return platform
	}
}
