package cascade_test

import (
	"context"
	"github.com/Peripli/service-manager/operations"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/cascade"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("cascade operations", func() {
	Context("children query", func() {

		JustBeforeEach(func() {
			initTenantResources(true, true)
		})

		It("find tenant cascade children", func() {
			tenantResource := types.NewTenant(tenantID, multitenancyLabel)
			tenantCascade := &cascade.TenantCascade{
				Tenant: tenantResource,
			}
			children, err := operations.ListCascadeChildren(context.TODO(), tenantCascade.GetChildrenCriterion(), ctx.SMRepository)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(children[types.PlatformType].Len()).To(BeIdenticalTo(1))
			Expect(children[types.ServiceBrokerType].Len()).To(BeIdenticalTo(1))
			Expect(children[types.ServiceInstanceType].Len()).To(BeIdenticalTo(8))
			Expect(children[types.PlatformType].ItemAt(0).GetID()).To(BeIdenticalTo(platformID))
			Expect(children[types.ServiceBrokerType].ItemAt(0).GetID()).To(BeIdenticalTo(tenantBrokerID))
			var instanceIDs []string
			for i := 0; i < children[types.ServiceInstanceType].Len(); i++ {
				instanceIDs = append(instanceIDs, children[types.ServiceInstanceType].ItemAt(i).GetID())
			}
			Expect(instanceIDs).To(ConsistOf([]string{
				globalPlatformGlobalBrokerInstanceID,
				globalPlatformTenantBrokerInstanceID,
				tenantPlatformGlobalBrokerInstanceID,
				osbInstanceID,
				smaapInstanceID1,
				smaapInstanceID2,
				sharedInstanceID,
				referenceInstanceID,
			}))
		})
		It("find platform cascade children", func() {
			platformObj, err := ctx.SMRepository.Get(context.Background(), types.PlatformType, query.ByField(query.EqualsOperator, "id", platformID))
			Expect(err).NotTo(HaveOccurred())
			platformResource := platformObj.(*types.Platform)
			Expect(err).ToNot(HaveOccurred())
			platformCascade := &cascade.PlatformCascade{
				Platform: platformResource,
			}
			children, err := operations.ListCascadeChildren(context.TODO(), platformCascade.GetChildrenCriterion(), ctx.SMRepository)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(children[types.ServiceInstanceType].Len()).To(BeIdenticalTo(3))
			var instanceIDs []string
			for i := 0; i < children[types.ServiceInstanceType].Len(); i++ {
				instanceIDs = append(instanceIDs, children[types.ServiceInstanceType].ItemAt(i).GetID())
			}
			Expect(instanceIDs).To(ConsistOf([]string{
				tenantPlatformGlobalBrokerInstanceID,
				osbInstanceID,
				sharedInstanceID,
			}))
		})
		It("find service broker cascade children", func() {
			brokerObj, err := ctx.SMRepository.Get(context.Background(), types.ServiceBrokerType, query.ByField(query.EqualsOperator, "id", tenantBrokerID))
			Expect(err).NotTo(HaveOccurred())
			err = operations.EnrichBrokersOfferings(context.Background(), brokerObj, ctx.SMRepository)
			Expect(err).NotTo(HaveOccurred())
			brokerResource := brokerObj.(*types.ServiceBroker)
			Expect(err).ToNot(HaveOccurred())
			brokerCascade := &cascade.ServiceBrokerCascade{
				ServiceBroker: brokerResource,
			}
			children, err := operations.ListCascadeChildren(context.TODO(), brokerCascade.GetChildrenCriterion(), ctx.SMRepository)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(children[types.ServiceInstanceType].Len()).To(BeIdenticalTo(5))
			var instanceIDs []string
			for i := 0; i < children[types.ServiceInstanceType].Len(); i++ {
				instanceIDs = append(instanceIDs, children[types.ServiceInstanceType].ItemAt(i).GetID())
			}
			Expect(instanceIDs).To(ConsistOf([]string{
				globalPlatformTenantBrokerInstanceID,
				smaapInstanceID1,
				osbInstanceID,
				sharedInstanceID,
				referenceInstanceID,
			}))
		})
		It("find service instance without bindings cascade children", func() {
			instanceObj, err := ctx.SMRepository.Get(context.Background(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "id", smaapInstanceID2))
			Expect(err).NotTo(HaveOccurred())
			instanceResource := instanceObj.(*types.ServiceInstance)
			Expect(err).ToNot(HaveOccurred())
			instanceCascade := &cascade.ServiceInstanceCascade{
				ServiceInstance: instanceResource,
			}
			children, err := operations.ListCascadeChildren(context.TODO(), instanceCascade.GetChildrenCriterion(), ctx.SMRepository)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(children[types.ServiceInstanceType].Len()).To(BeIdenticalTo(0))
			Expect(children[types.ServiceBindingType].Len()).To(BeIdenticalTo(0))
		})
		It("find service instance with bindings cascade children", func() {
			instanceObj, err := ctx.SMRepository.Get(context.Background(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "id", osbInstanceID))
			Expect(err).NotTo(HaveOccurred())
			instanceResource := instanceObj.(*types.ServiceInstance)
			Expect(err).ToNot(HaveOccurred())
			instanceCascade := &cascade.ServiceInstanceCascade{
				ServiceInstance: instanceResource,
			}
			children, err := operations.ListCascadeChildren(context.TODO(), instanceCascade.GetChildrenCriterion(), ctx.SMRepository)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(children[types.ServiceInstanceType].Len()).To(BeIdenticalTo(0))
			Expect(children[types.ServiceBindingType].Len()).To(BeIdenticalTo(2))
			var bindingIDs []string
			for i := 0; i < children[types.ServiceBindingType].Len(); i++ {
				bindingIDs = append(bindingIDs, children[types.ServiceBindingType].ItemAt(i).GetID())
			}
			Expect(bindingIDs).To(ConsistOf([]string{
				osbBindingID1,
				osbBindingID2,
			}))
		})
		It("find service instance with shared references cascade children", func() {
			instanceObj, err := ctx.SMRepository.Get(context.Background(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "id", sharedInstanceID))
			Expect(err).NotTo(HaveOccurred())
			instanceResource := instanceObj.(*types.ServiceInstance)
			Expect(err).ToNot(HaveOccurred())
			instanceCascade := &cascade.ServiceInstanceCascade{
				ServiceInstance: instanceResource,
			}
			children, err := operations.ListCascadeChildren(context.TODO(), instanceCascade.GetChildrenCriterion(), ctx.SMRepository)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(children[types.ServiceBindingType].Len()).To(BeIdenticalTo(0))
			Expect(children[types.ServiceInstanceType].Len()).To(BeIdenticalTo(1))
			var instanceIDs []string
			for i := 0; i < children[types.ServiceInstanceType].Len(); i++ {
				instanceIDs = append(instanceIDs, children[types.ServiceInstanceType].ItemAt(i).GetID())
			}
			Expect(instanceIDs).To(ConsistOf([]string{
				referenceInstanceID,
			}))
		})
	})
})
