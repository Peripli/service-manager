/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// TODO
// test0: register a broker with one free plan and one paid plan, verify the free plan and the paid plan are created. Verify a public visibility is created for the free plan and none is created for the paid plan.
// test1: add new free plan, update broker, verify plan is created, verify public visibility for the plan is created
// test2: add new paid plan, update broker, verify no public visibility is created for the plan
// test3: verify public visibility exists for the existing free plan, make existing free plan paid, update broker, verify public visibility is no longer present
// test4: verify public visibility does not exist for the existing paid plan, make existing paid plan free, update broker, verify public visibility is created

package filter_test

import (
	"testing"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/test/common"
)
import . "github.com/onsi/ginkgo"
import . "github.com/onsi/gomega"

func TestFreePlansFilter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Free Plans Filter Tests Suite")
}

const testCatalog = ``

var _ = Describe("Service Manager Filters", func() {

	verifyServicePlansArePresent := func() {

	}

	verifyVisibilityIsPresent := func() {

	}

	verifyVisibilityIsNotPresent := func() {

	}

	getServicePlanFromCatalog := func(catalog string) *types.ServicePlan {

	}

	var ctx *common.TestContext
	var existingBrokerID string
	var existingPlatformID string
	var existingBrokerServer *common.BrokerServer

	BeforeSuite(func() {
		ctx = common.NewTestContext(nil)
		existingBrokerID, existingBrokerServer = ctx.RegisterBrokerWithCatalog(testCatalog)
		Expect(existingBrokerID).ToNot(BeEmpty())

		platform := ctx.TestPlatform
		existingPlatformID = platform.ID
		Expect(existingPlatformID).ToNot(BeEmpty())
	})

	BeforeEach(func() {
		existingBrokerServer.Reset()
		common.RemoveAllBrokers(ctx.SMWithOAuth)
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

})
