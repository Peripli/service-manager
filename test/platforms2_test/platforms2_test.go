package platforms2_test

import (
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test"
	"github.com/Peripli/service-manager/test/common"
)

func TestPlatforms(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Platforms Tests Suite")

}

var _ = test.DescribeTestsFor(test.TestCase{
	API: "platforms",
	//Prerequisites: {
	//	[]resource: {
	//		name: "service_offerings",
	//		reference: "service_offerings_id",
	//		required: true,
	//	},
	//},
	Op: []string{"list", "get"},
	RandomResourceObjectGenerator: func(ctx *common.TestContext) common.Object {
		platform := ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(common.GenerateRandomPlatform()).
			Expect().
			Status(http.StatusCreated).JSON().Object().Raw()
		delete(platform, "credentials")
		return platform
	},
},
)
