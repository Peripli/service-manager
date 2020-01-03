/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package operation_test

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"

	"fmt"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"

	"github.com/Peripli/service-manager/test"

	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

func TestOperations(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operations Tests Suite")
}

var _ = test.DescribeTestsFor(test.TestCase{
	API: web.OperationsURL,
	SupportedOps: []test.Op{
		test.Get, test.List,
	},
	ResourceType: types.OperationType,
	MultitenancySettings: &test.MultitenancySettings{
		ClientID:           "tenancyClient",
		ClientIDTokenClaim: "cid",
		TenantTokenClaim:   "zid",
		LabelKey:           "tenant",
		TokenClaims: map[string]interface{}{
			"cid": "tenancyClient",
			"zid": "tenantID",
		},
	},
	ResourceBlueprint:                      blueprint,
	ResourceWithoutNullableFieldsBlueprint: blueprint,
	PatchResource: func(ctx *common.TestContext, apiPath string, objID string, resourceType types.ObjectType, patchLabels []*query.LabelChange) {
		byID := query.ByField(query.EqualsOperator, "id", objID)
		si, err := ctx.SMRepository.Get(context.Background(), resourceType, byID)
		if err != nil {
			Fail(fmt.Sprintf("unable to retrieve resource %s: %s", resourceType, err))
		}

		_, err = ctx.SMRepository.Update(context.Background(), si, patchLabels)
		if err != nil {
			Fail(fmt.Sprintf("unable to update resource %s: %s", resourceType, err))
		}
	},
})

func blueprint(ctx *common.TestContext, auth *common.SMExpect) common.Object {
	id, err := uuid.NewV4()
	if err != nil {
		Fail(err.Error())
	}
	labels := make(map[string][]string)
	if ctx.SMWithOAuthForTenant == auth {
		labels["tenant"] = append([]string{"tenantID"})
	}
	_, err = ctx.SMRepository.Create(context.TODO(), &types.Operation{
		Base: types.Base{
			ID:        id.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Labels:    labels,
		},
		Description:   "test",
		Type:          types.CREATE,
		State:         types.IN_PROGRESS,
		ResourceID:    id.String(),
		ResourceType:  "/v1/service_brokers",
		CorrelationID: id.String(),
	})
	if err != nil {
		Fail(err.Error())
	}
	return auth.ListWithQuery(web.OperationsURL, fmt.Sprintf("fieldQuery=id eq '%s'", id)).First().Object().Raw()
}
