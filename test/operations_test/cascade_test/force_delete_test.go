package cascade_test

import (
	"context"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/cascade"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("cascade force delete", func() {
	Context("force delete instances", func() {
		JustBeforeEach(func() {
			initTenantResources(true)
		})

		It("should succeed: delete instance using force", func() {
			globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
				return http.StatusAccepted, common.Object{"async": true}
			})
			registerInstanceLastOPHandlers(globalBrokerServer, http.StatusInternalServerError, types.FAILED)
			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*3+pollCascade*3).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			validateResourcesDeleted(ctx.SMRepository, fullTree.byResourceType)
			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsCopied(fullTree)

			By("validating tenant error is a collection of his child errors")
			tenantOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot(rootID))
			Expect(err).NotTo(HaveOccurred())

			errors := cascade.CascadeErrors{}
			err = json.Unmarshal(tenantOP.(*types.Operation).Errors, &errors)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(errors.Errors)).To(Equal(3))

			for _, e := range errors.Errors {
				Expect(e.ParentID).To(Or(Equal(platformID), Equal(tenantBrokerID), Equal(tenantID)))
				Expect(len(e.Message)).Should(BeNumerically(">", 0))
				Expect(e.ResourceType).To(Equal(types.ServiceInstanceType))
				Expect(e.ParentType).To(Or(Equal(types.ServiceBrokerType), Equal(types.PlatformType), Equal(types.TenantType)))
			}
		})

		It("should succeed: delete container using force", func() {
			// tree contains this branch: sa(root) -> platform -> instance(container) -> instance(in container) -> binding
			container := createContainerWithChildren()
			// binding under container will have failed state in operation, failure will propagate up through the container as well
			registerBindingLastOPHandlers(tenantBrokerServer, http.StatusInternalServerError, types.FAILED)
			registerBindingLastOPHandlers(globalBrokerServer, http.StatusInternalServerError, types.FAILED)

			newCtx := context.WithValue(context.Background(), cascade.ParentInstanceLabelKey{}, "containerID")
			rootID := triggerCascadeOperation(newCtx, types.TenantType, tenantID, true)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*3+pollCascade*3).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			validateResourcesDeleted(ctx.SMRepository, fullTree.byResourceType)
			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsCopied(fullTree)

			By("validating tenant error is a collection of his child errors")
			tenantOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot(rootID))
			Expect(err).NotTo(HaveOccurred())

			errors := cascade.CascadeErrors{}
			err = json.Unmarshal(tenantOP.(*types.Operation).Errors, &errors)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(errors.Errors)).To(Equal(3))

			instanceInContainer := container.instances[0]
			bindingForInstanceInContainer := container.bindingForInstance[instanceInContainer][0]

			for _, e := range errors.Errors {
				Expect(e.ParentID).To(Or(Equal(osbInstanceID), Equal(instanceInContainer)))
				Expect(e.ResourceID).To(Or(Equal("binding1"), Equal("binding2"), Equal(bindingForInstanceInContainer)))
				Expect(len(e.Message)).Should(BeNumerically(">", 0))
				Expect(e.ResourceType).To(Equal(types.ServiceBindingType))
				Expect(e.ParentType).To(Equal(types.ServiceInstanceType))
			}
		})
	})

	Context("force delete binding", func() {
		JustBeforeEach(func() {
			initTenantResources(true)
		})
		It("should succeed: delete binding using force", func() {
			registerBindingLastOPHandlers(tenantBrokerServer, http.StatusInternalServerError, types.FAILED)
			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*3+pollCascade*3).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			validateResourcesDeleted(ctx.SMRepository, fullTree.byResourceType)
			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsCopied(fullTree)

			By("validating tenant error is a collection of his child errors")
			tenantOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot(rootID))
			Expect(err).NotTo(HaveOccurred())

			errors := cascade.CascadeErrors{}
			err = json.Unmarshal(tenantOP.(*types.Operation).Errors, &errors)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(errors.Errors)).To(Equal(2))

			for _, e := range errors.Errors {
				Expect(e.ParentID).To(Equal(osbInstanceID))
				Expect(e.ResourceID).To(Or(Equal("binding1"), Equal("binding2")))
				Expect(len(e.Message)).Should(BeNumerically(">", 0))
				Expect(e.ResourceType).To(Equal(types.ServiceBindingType))
				Expect(e.ParentType).To(Equal(types.ServiceInstanceType))
			}
		})

		It("should fail: delete binding using force", func() {
			// need to check that errors include db err

			registerBindingLastOPHandlers(tenantBrokerServer, http.StatusInternalServerError, types.FAILED)
			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)
			createOSBBinding(ctx, ctx.SMWithBasic, tenantBrokerID, osbInstanceID, "binding3", map[string]interface{}{
				"service_id":        "test-service",
				"plan_id":           "plan-service",
				"organization_guid": "my-org",
			})

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					queryFailures)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*5+pollCascade*5).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsCopied(fullTree)

			By("validating tenant error is a collection of his child errors")
			tenantOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot(rootID))
			Expect(err).NotTo(HaveOccurred())

			errors := cascade.CascadeErrors{}
			err = json.Unmarshal(tenantOP.(*types.Operation).Errors, &errors)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(errors.Errors)).To(Equal(3))

			for _, e := range errors.Errors {
				Expect(e.ParentID).To(Or(Equal(osbInstanceID), Equal("")))
				Expect(e.ResourceID).To(Or(Equal("binding1"), Equal("binding2"), Equal("tenant_value")))
				Expect(len(e.Message)).Should(BeNumerically(">", 0))
				Expect(e.ResourceType).To(Or(Equal(types.ServiceBindingType), Equal(types.TenantType)))
			}
		})

	})

	Context("force delete with direct instance children only", func() {
		JustBeforeEach(func() {
			initTenantResources(false)
			ctx.SMWithBasic.SetBasicCredentials(ctx, ctx.TestPlatform.Credentials.Basic.Username, ctx.TestPlatform.Credentials.Basic.Password)
			createOSBInstance(ctx, ctx.SMWithBasic, globalBrokerID, osbInstanceID, map[string]interface{}{
				"service_id":        "global-service",
				"plan_id":           "global-plan",
				"organization_guid": "my-orgafsf",
				"context": map[string]string{
					"tenant": tenantID,
				},
			})
		})

		It("should succeed: delete tenant with only direct instance children", func() {
			globalBrokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", func(req *http.Request) (int, map[string]interface{}) {
				return http.StatusBadRequest, common.Object{}
			})
			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					querySucceeded)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*5+pollCascade*5).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			AssertOperationCount(func(count int) {
				Expect(count).To(Equal(1))
			}, queryForInstanceOperations, queryFailures)

			validateResourcesDeleted(ctx.SMRepository, fullTree.byResourceType)
			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsCopied(fullTree)

			By("validating tenant error is a collection of his child errors")
			tenantOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot(rootID))
			Expect(err).NotTo(HaveOccurred())

			errors := cascade.CascadeErrors{}
			err = json.Unmarshal(tenantOP.(*types.Operation).Errors, &errors)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(errors.Errors)).To(Equal(1))
			e := errors.Errors[0]
			Expect(e.ParentID).To(Equal("tenant_value"))
			Expect(e.ResourceID).To(Equal(osbInstanceID))
			Expect(len(e.Message)).Should(BeNumerically(">", 0))
			Expect(e.ResourceType).To(Equal(types.ServiceInstanceType))
		})

		It("should fail: delete tenant with only direct instance children", func() {
			rootID := triggerCascadeOperation(context.Background(), types.TenantType, tenantID, true)
			createOSBBinding(ctx, ctx.SMWithBasic, tenantBrokerID, osbInstanceID, "binding3", map[string]interface{}{
				"service_id":        "global-service",
				"plan_id":           "global-plan",
				"organization_guid": "my-org",
			})

			By("waiting cascading process to finish")
			Eventually(func() int {
				count, err := ctx.SMRepository.Count(
					context.Background(),
					types.OperationType,
					queryForRoot(rootID),
					queryFailures)
				Expect(err).NotTo(HaveOccurred())

				return count
			}, actionTimeout*5+pollCascade*5).Should(Equal(1))

			fullTree, err := fetchFullTree(ctx.SMRepository, rootID)
			Expect(err).ToNot(HaveOccurred())

			AssertOperationCount(func(count int) {
				Expect(count).To(Equal(1))
			}, queryForInstanceOperations, queryFailures)

			validateParentsRanAfterChildren(fullTree)
			validateDuplicationsCopied(fullTree)

			By("validating tenant error is a collection of his child errors")
			tenantOP, err := ctx.SMRepository.Get(context.Background(), types.OperationType, queryForRoot(rootID))
			Expect(err).NotTo(HaveOccurred())

			errors := cascade.CascadeErrors{}
			err = json.Unmarshal(tenantOP.(*types.Operation).Errors, &errors)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(errors.Errors)).To(Equal(2))

			for _, e := range errors.Errors {
				Expect(e.ParentID).To(Or(Equal("tenant_value"), Equal("")))
				Expect(e.ResourceID).To(Or(Equal("tenant_value"), Equal("test-instance")))
				Expect(len(e.Message)).Should(BeNumerically(">", 0))
				Expect(e.ResourceType).To(Or(Equal(types.ServiceInstanceType), Equal(types.TenantType)))
			}
		})
	})
})
