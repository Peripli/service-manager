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

package test

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"github.com/Peripli/service-manager/pkg/util"

	"time"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/gavv/httpexpect"
	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

type Op string

type ResponseMode bool

const (
	Get        Op = "get"
	List       Op = "list"
	Delete     Op = "delete"
	DeleteList Op = "deletelist"
	Patch      Op = "patch"

	Sync  ResponseMode = false
	Async ResponseMode = true
)

type MultitenancySettings struct {
	ClientID           string
	ClientIDTokenClaim string
	TenantTokenClaim   string
	LabelKey           string

	TokenClaims map[string]interface{}
}

type TestCase struct {
	API                             string
	SupportsAsyncOperations         bool
	SupportsCascadeDeleteOperations bool
	SupportedOps                    []Op
	ResourceType                    types.ObjectType
	ResourcePropertiesToIgnore      []string

	MultitenancySettings   *MultitenancySettings
	DisableTenantResources bool
	StrictlyTenantScoped   bool
	DisableBasicAuth       bool

	ResourceBlueprint                      func(ctx *common.TestContext, smClient *common.SMExpect, async bool) common.Object
	ResourceWithoutNullableFieldsBlueprint func(ctx *common.TestContext, smClient *common.SMExpect, async bool) common.Object
	PatchResource                          func(ctx *common.TestContext, tenantScoped bool, apiPath string, objID string, resourceType types.ObjectType, patchLabels []*types.LabelChange, async bool)
	SubResourcesBlueprint                  func(ctx *common.TestContext, smClient *common.SMExpect, async bool, resourceID string, resourceType types.ObjectType, resource common.Object)

	AdditionalTests func(ctx *common.TestContext, t *TestCase)
	ContextBuilder  *common.TestContextBuilder
}

func stripObject(obj common.Object, properties ...string) {
	delete(obj, "created_at")
	delete(obj, "updated_at")

	for _, prop := range properties {
		delete(obj, prop)
	}
}

func APIResourcePatch(ctx *common.TestContext, tenantScoped bool, apiPath string, objID string, objType types.ObjectType, patchLabels []*types.LabelChange, async bool) {
	patchLabelsBody := make(map[string]interface{})
	patchLabelsBody["labels"] = patchLabels

	By(fmt.Sprintf("Attempting to patch resource of %s with labels as labels are declared supported", apiPath))
	var resp *httpexpect.Response

	if tenantScoped {
		resp = ctx.SMWithOAuthForTenant.PATCH(apiPath+"/"+objID).WithQuery("async", strconv.FormatBool(async)).WithJSON(patchLabelsBody).Expect()
	} else {
		resp = ctx.SMWithOAuth.PATCH(apiPath+"/"+objID).WithQuery("async", strconv.FormatBool(async)).WithJSON(patchLabelsBody).Expect()
	}

	if async {
		resp = resp.Status(http.StatusAccepted)
	} else {
		resp.Status(http.StatusOK)
	}

	common.VerifyOperationExists(ctx, resp.Header("Location").Raw(), common.OperationExpectations{
		Category:          types.UPDATE,
		State:             types.SUCCEEDED,
		ResourceType:      objType,
		Reschedulable:     false,
		DeletionScheduled: false,
	})
}

func StorageResourcePatch(ctx *common.TestContext, _ bool, _ string, objID string, resourceType types.ObjectType, patchLabels []*types.LabelChange, _ bool) {
	byID := query.ByField(query.EqualsOperator, "id", objID)
	sb, err := ctx.SMRepository.Get(context.Background(), resourceType, byID)
	if err != nil {
		Fail(fmt.Sprintf("unable to retrieve resource %s: %s", resourceType, err))
	}

	_, err = ctx.SMRepository.Update(context.Background(), sb, patchLabels)
	if err != nil {
		Fail(fmt.Sprintf("unable to update resource %s: %s", resourceType, err))
	}
}

func EnsurePublicPlanVisibility(repository storage.Repository, planID string) {
	EnsurePublicPlanVisibilityForPlatform(repository, planID, "")
}

func EnsurePublicPlanVisibilityForPlatform(repository storage.Repository, planID, platformID string) {
	EnsurePlanVisibility(repository, "", platformID, planID, "")
}

func EnsurePlanVisibilityDoesNotExist(repository storage.Repository, tenantIdentifier, platformID, planID, tenantID string) {
	var criteria []query.Criterion

	if planID != "" {
		criteria = append(criteria, query.ByField(query.EqualsOperator, "service_plan_id", planID))
	}
	if platformID != "" {
		criteria = append(criteria, query.ByField(query.EqualsOperator, "platform_id", platformID))
	}
	if tenantIdentifier != "" {
		criteria = append(criteria, query.ByLabel(query.EqualsOperator, tenantIdentifier, tenantID))
	}

	if err := repository.Delete(context.TODO(), types.VisibilityType, criteria...); err != nil {
		if err != util.ErrNotFoundInStorage {
			panic(err)
		}
	}
}

func EnsurePlanVisibility(repository storage.Repository, tenantIdentifier, platformID, planID, tenantID string) {
	EnsurePlanVisibilityDoesNotExist(repository, tenantIdentifier, platformID, planID, tenantID)
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Errorf("could not generate GUID for visibility: %s", err))
	}

	var labels types.Labels = nil
	if tenantID != "" {
		labels = types.Labels{
			tenantIdentifier: {tenantID},
		}
	}
	currentTime := time.Now().UTC()
	_, err = repository.Create(context.TODO(), &types.Visibility{
		Base: types.Base{
			ID:        UUID.String(),
			UpdatedAt: currentTime,
			CreatedAt: currentTime,
			Labels:    labels,
			Ready:     true,
		},
		ServicePlanID: planID,
		PlatformID:    platformID,
	})
	if err != nil {
		panic(err)
	}
}

func DescribeTestsFor(t TestCase) bool {
	return Describe(t.API, func() {
		var ctx *common.TestContext

		AfterSuite(func() {
			ctx.Cleanup()
		})

		ctxBuilder := func() *common.TestContextBuilder {
			ctxBuilder := common.NewTestContextBuilderWithSecurity()

			if t.MultitenancySettings != nil {
				ctxBuilder.
					WithTenantTokenClaims(t.MultitenancySettings.TokenClaims).
					WithEnvPostExtensions(func(e env.Environment, servers map[string]common.FakeServer) {
						e.Set("api.protected_labels", t.MultitenancySettings.LabelKey)
					}).
					WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
						_, err := smb.EnableMultitenancy(t.MultitenancySettings.LabelKey, common.ExtractTenantFunc)
						return err
					})
			}
			return ctxBuilder
		}

		BeforeEach(func() {
			t.ContextBuilder = ctxBuilder()
		})

		func() {
			By("==== Preparation for SM tests... ====")

			defer GinkgoRecover()
			ctx = ctxBuilder().Build()

			// A panic outside of Ginkgo's primitives (during test setup) would be recovered
			// by the deferred GinkgoRecover() and the error will be associated with the first
			// It to be ran in the suite. There, we add a dummy It to reduce confusion.
			It("sets up all test prerequisites that are ran outside of Ginkgo primitives properly", func() {
				Expect(true).To(BeTrue())
			})

			responseModes := []ResponseMode{Sync}
			if t.SupportsAsyncOperations {
				responseModes = append(responseModes, Async)
			}

			for _, op := range t.SupportedOps {
				for _, respMode := range responseModes {
					switch op {
					case Get:
						DescribeGetTestsfor(ctx, t, respMode)
					case List:
						DescribeListTestsFor(ctx, t, respMode)
					case Delete:
						DescribeDeleteTestsfor(ctx, t, respMode, t.SupportsCascadeDeleteOperations)
					case DeleteList:
						if respMode == Sync {
							DescribeDeleteListFor(ctx, t)
						}
					case Patch:
						DescribePatchTestsFor(ctx, t, respMode)
					default:
						_, err := fmt.Fprintf(GinkgoWriter, "Generic test cases for op %s are not implemented\n", op)
						if err != nil {
							panic(err)
						}
					}
				}
			}

			if t.AdditionalTests != nil {
				t.AdditionalTests(ctx, &t)
			}

			By("==== Successfully finished preparation for SM tests. Running API tests suite... ====")
		}()
	})
}

func RegisterBrokerPlatformCredentialsExpect(SMBasicPlatform *common.SMExpect, brokerID string, expectedStatusCode int) (string, string) {
	return RegisterBrokerPlatformCredentialsWithNotificationIDExpect(SMBasicPlatform, brokerID, "", expectedStatusCode)
}

func RegisterBrokerPlatformCredentials(SMBasicPlatform *common.SMExpect, brokerID string) (string, string) {
	return RegisterBrokerPlatformCredentialsWithNotificationID(SMBasicPlatform, brokerID, "")
}

func RegisterBrokerPlatformCredentialsWithNotificationID(SMBasicPlatform *common.SMExpect, brokerID, notificationID string) (string, string) {
	return RegisterBrokerPlatformCredentialsWithNotificationIDExpect(SMBasicPlatform, brokerID, notificationID, http.StatusOK)
}

func RegisterBrokerPlatformCredentialsWithNotificationIDExpect(SMBasicPlatform *common.SMExpect, brokerID, notificationID string, expectedStatusCode int) (string, string) {
	username, password, payload := getBrokerPlatformCredentialsPayload(brokerID, notificationID)

	res := SMBasicPlatform.Request(http.MethodPut, web.BrokerPlatformCredentialsURL).
		WithJSON(payload).Expect().Status(expectedStatusCode).JSON().Object()

	if expectedStatusCode == http.StatusOK {
		SMBasicPlatform.Request(http.MethodPut, fmt.Sprintf("%s/%s/activate", web.BrokerPlatformCredentialsURL, res.Value("id").String().Raw())).
			WithJSON(common.Object{}).Expect().Status(http.StatusOK)
	}
	return username, password
}

func RegisterBrokerPlatformCredentialsWithNotificationIDNoActivateExpect(SMBasicPlatform *common.SMExpect, brokerID, notificationID string, expectedStatusCode int) (string, string) {
	username, password, payload := getBrokerPlatformCredentialsPayload(brokerID, notificationID)

	SMBasicPlatform.Request(http.MethodPut, web.BrokerPlatformCredentialsURL).
		WithJSON(payload).Expect().Status(expectedStatusCode)

	return username, password
}

func getBrokerPlatformCredentialsPayload(brokerID string, notificationID string) (string, string, map[string]interface{}) {
	username, err := util.GenerateCredential()
	Expect(err).ToNot(HaveOccurred())
	password, err := util.GenerateCredential()
	Expect(err).ToNot(HaveOccurred())

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	payload := map[string]interface{}{
		"broker_id":       brokerID,
		"username":        username,
		"password_hash":   string(passwordHash),
		"notification_id": notificationID,
	}
	return username, password, payload
}
