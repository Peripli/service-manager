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

package reconcile_test

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/agent/reconcile"

	"github.com/Peripli/service-manager/pkg/agent/platform"
	"github.com/Peripli/service-manager/pkg/agent/platform/platformfakes"
	"github.com/Peripli/service-manager/pkg/agent/sm/smfakes"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconcile brokers", func() {

	var (
		fakeSMClient *smfakes.FakeClient

		fakePlatformCatalogFetcher     *platformfakes.FakeCatalogFetcher
		fakePlatformBrokerClient       *platformfakes.FakeBrokerClient
		fakePlatformVisibilitiesClient *platformfakes.FakeVisibilityClient

		reconcileSettings *reconcile.Settings
		reconciler        *reconcile.Reconciler

		smbroker1      *types.ServiceBroker
		smbroker2      *types.ServiceBroker
		smbroker3      *types.ServiceBroker
		smbroker4      *types.ServiceBroker
		smOrphanBroker *types.ServiceBroker

		platformbroker1                  *platform.ServiceBroker
		platformbroker2                  *platform.ServiceBroker
		platformbrokerNonProxy           *platform.ServiceBroker
		platformbrokerNonProxy2          *platform.ServiceBroker
		platformBrokerProxy              *platform.ServiceBroker
		platformBrokerProxy2             *platform.ServiceBroker
		platformOrphanBrokerProxy        *platform.ServiceBroker
		platformOrphanBrokerProxyRenamed *platform.ServiceBroker
	)

	stubCreateBrokerToSucceed := func(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
		return &platform.ServiceBroker{
			GUID:      r.Name,
			Name:      r.Name,
			BrokerURL: r.BrokerURL,
		}, nil
	}

	stubCreateBrokerToReturnError := func(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
		return nil, fmt.Errorf("error")
	}

	stubPlatformOpsToSucceed := func() {
		fakePlatformBrokerClient.CreateBrokerStub = stubCreateBrokerToSucceed
		fakePlatformBrokerClient.DeleteBrokerReturns(nil)
		fakePlatformCatalogFetcher.FetchReturns(nil)
	}

	stubPlatformUpdateBroker := func(broker *platform.ServiceBroker) {
		fakePlatformBrokerClient.UpdateBrokerReturns(broker, nil)
	}

	BeforeEach(func() {
		fakeSMClient = &smfakes.FakeClient{}
		fakePlatformClient := &platformfakes.FakeClient{}

		fakePlatformBrokerClient = &platformfakes.FakeBrokerClient{}
		fakePlatformCatalogFetcher = &platformfakes.FakeCatalogFetcher{}
		fakePlatformVisibilitiesClient = &platformfakes.FakeVisibilityClient{}

		fakePlatformClient.BrokerReturns(fakePlatformBrokerClient)
		fakePlatformClient.CatalogFetcherReturns(fakePlatformCatalogFetcher)
		fakePlatformClient.VisibilityReturns(fakePlatformVisibilitiesClient)

		platformClient := struct {
			*platformfakes.FakeCatalogFetcher
			*platformfakes.FakeClient
		}{
			FakeCatalogFetcher: fakePlatformCatalogFetcher,
			FakeClient:         fakePlatformClient,
		}

		reconcileSettings = reconcile.DefaultSettings()
		reconciler = &reconcile.Reconciler{
			Resyncer: reconcile.NewResyncer(reconcileSettings, platformClient, fakeSMClient, fakeSMAppHost, fakeProxyPathPattern),
		}

		smbroker1 = &types.ServiceBroker{
			Base: types.Base{
				ID: "smBrokerID1",
			},
			Name:      "smBroker1",
			BrokerURL: "https://smBroker1.com",
		}

		smbroker2 = &types.ServiceBroker{
			Base: types.Base{
				ID: "smBrokerID2",
			},
			Name:      "smBroker2",
			BrokerURL: "https://smBroker2.com",
		}

		platformbrokerNonProxy = &platform.ServiceBroker{
			GUID:      "platformBrokerID3",
			Name:      "platformBroker3",
			BrokerURL: "https://platformBroker3.com",
		}

		platformbrokerNonProxy2 = &platform.ServiceBroker{
			GUID:      "platformBrokerID4",
			Name:      "platformBroker4",
			BrokerURL: "https://platformBroker4.com",
		}

		smbroker3 = &types.ServiceBroker{
			Base: types.Base{
				ID: "smBrokerID3",
			},
			Name:      platformbrokerNonProxy.Name,
			BrokerURL: platformbrokerNonProxy.BrokerURL,
		}

		smbroker4 = &types.ServiceBroker{
			Base: types.Base{
				ID: "smBrokerID4",
			},
			Name:      platformbrokerNonProxy2.Name,
			BrokerURL: platformbrokerNonProxy2.BrokerURL,
		}

		platformbroker1 = &platform.ServiceBroker{
			GUID:      "platformBrokerID1",
			Name:      brokerProxyName(smbroker1.Name, smbroker1.ID),
			BrokerURL: fakeSMAppHost + "/" + smbroker1.ID,
		}

		platformbroker2 = &platform.ServiceBroker{
			GUID:      "platformBrokerID2",
			Name:      brokerProxyName(smbroker2.Name, smbroker2.ID),
			BrokerURL: fakeSMAppHost + "/" + smbroker2.ID,
		}

		platformBrokerProxy = &platform.ServiceBroker{
			GUID:      platformbrokerNonProxy.GUID,
			Name:      brokerProxyName(smbroker3.Name, smbroker3.ID),
			BrokerURL: fakeSMAppHost + "/" + smbroker3.ID,
		}

		smOrphanBroker = &types.ServiceBroker{
			Base: types.Base{
				ID: "orphanBrokerProxyID",
			},
			Name:      "orphanBrokerProxy",
			BrokerURL: "https://orphanbroker.com",
		}

		platformOrphanBrokerProxy = &platform.ServiceBroker{
			GUID:      "platformOrphanBrokerProxy",
			Name:      brokerProxyName(smOrphanBroker.Name, smOrphanBroker.ID),
			BrokerURL: fakeProxyAppHost + "/v1/osb/" + smOrphanBroker.ID,
		}

		platformOrphanBrokerProxyRenamed = &platform.ServiceBroker{
			GUID:      "platformOrphanBrokerProxy",
			Name:      "test",
			BrokerURL: fakeProxyAppHost + "/v1/osb/" + smOrphanBroker.ID,
		}

		platformBrokerProxy2 = &platform.ServiceBroker{
			GUID:      platformOrphanBrokerProxy.GUID,
			Name:      brokerProxyName(smOrphanBroker.Name, smOrphanBroker.ID),
			BrokerURL: fakeSMAppHost + "/" + smOrphanBroker.ID,
		}
	})

	type expectations struct {
		reconcileCreateCalledFor  []*platform.ServiceBroker
		reconcileDeleteCalledFor  []*platform.ServiceBroker
		reconcileCatalogCalledFor []*platform.ServiceBroker
		reconcileUpdateCalledFor  []*platform.ServiceBroker
	}

	type testCase struct {
		stubs           func()
		platformBrokers func() ([]*platform.ServiceBroker, error)
		smBrokers       func() ([]*types.ServiceBroker, error)
		brokerBlacklist func() []string
		takeoverEnabled bool

		expectations func() expectations
	}

	entries := []TableEntry{
		Entry("When fetching brokers from SM fails no reconciliation should be done", testCase{
			stubs: func() {

			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return nil, fmt.Errorf("error fetching brokers")
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []*platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("When fetching brokers from platform fails no reconciliation should be done", testCase{
			stubs: func() {

			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return nil, fmt.Errorf("error fetching brokers")
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []*platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("When platform broker op fails reconciliation continues with the next broker", testCase{
			stubs: func() {
				fakePlatformBrokerClient.DeleteBrokerReturns(fmt.Errorf("error"))
				fakePlatformCatalogFetcher.FetchReturns(fmt.Errorf("error"))
				fakePlatformBrokerClient.CreateBrokerStub = stubCreateBrokerToReturnError
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformbroker2,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker1,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []*platform.ServiceBroker{
						platformbroker1,
					},
					reconcileDeleteCalledFor: []*platform.ServiceBroker{
						platformbroker2,
					},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("When broker from SM has no catalog reconciliation continues with the next broker", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformbroker1,
					platformbroker2,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker1,
					smbroker2,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []*platform.ServiceBroker{},
					reconcileDeleteCalledFor: []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{
						platformbroker1,
						platformbroker2,
					},
				}
			},
		}),

		Entry("When broker is in SM and is missing from platform it should be created", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker1,
					smbroker2,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []*platform.ServiceBroker{
						platformbroker1,
						platformbroker2,
					},
					reconcileDeleteCalledFor:  []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("When broker is in SM and is missing from platform and is part of the proxy broker blacklist it should not be created", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker1,
					smbroker2,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{smbroker1.Name}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []*platform.ServiceBroker{
						platformbroker2,
					},
					reconcileDeleteCalledFor:  []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("When all brokers in SM are missing from platform and are part of the proxy broker blacklist they should not be created", testCase{
			stubs: func() {},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker1,
					smbroker2,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{smbroker1.Name, smbroker2.Name}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []*platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("When broker is in SM and is also in platform it should be catalog refetched", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformbroker1,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker1,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []*platform.ServiceBroker{},
					reconcileDeleteCalledFor: []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{
						platformbroker1,
					},
				}
			},
		}),

		Entry("When broker is in SM and is also in platform but points to proxy URL it should be updated to point to SM URL", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
				stubPlatformUpdateBroker(platformBrokerProxy2)
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformOrphanBrokerProxy,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smOrphanBroker,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []*platform.ServiceBroker{},
					reconcileDeleteCalledFor: []*platform.ServiceBroker{},
					reconcileUpdateCalledFor: []*platform.ServiceBroker{
						platformBrokerProxy2,
					},
				}
			},
		}),

		Entry("When broker is in SM and is also in platform but points to proxy URL and was renamed in platform it should be updated to point to SM URL and name should be restored", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
				stubPlatformUpdateBroker(platformBrokerProxy2)
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformOrphanBrokerProxyRenamed,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smOrphanBroker,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []*platform.ServiceBroker{},
					reconcileDeleteCalledFor: []*platform.ServiceBroker{},
					reconcileUpdateCalledFor: []*platform.ServiceBroker{
						platformBrokerProxy2,
					},
				}
			},
		}),

		Entry("When broker is missing from SM but is in platform it should be deleted", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformbroker1,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []*platform.ServiceBroker{},
					reconcileDeleteCalledFor: []*platform.ServiceBroker{
						platformbroker1,
					},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("When broker is missing from SM but is in platform that is not represented by the proxy should be ignored", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformbrokerNonProxy,
					platformbrokerNonProxy2,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []*platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("When broker is registered in the platform and SM, but not yet taken over, it should be updated", testCase{
			stubs: func() {
				stubPlatformUpdateBroker(platformBrokerProxy)
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformbrokerNonProxy,
					platformbrokerNonProxy2,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker3,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []*platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
					reconcileUpdateCalledFor: []*platform.ServiceBroker{
						platformBrokerProxy,
					},
				}
			},
		}),

		Entry("When broker is registered in the platform and SM, but not yet taken over, but is also in the proxy broker blacklist it should be ignored", testCase{
			stubs: func() {},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformbrokerNonProxy,
					platformbrokerNonProxy2,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker3,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{smbroker3.Name}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []*platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
					reconcileUpdateCalledFor:  []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("When all brokers are registered in the platform and SM, but not yet taken over, but are also in the proxy broker blacklist they should be ignored", testCase{
			stubs: func() {},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformbrokerNonProxy,
					platformbrokerNonProxy2,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker3,
					smbroker4,
				}, nil
			},
			takeoverEnabled: true,
			brokerBlacklist: func() []string {
				return []string{smbroker3.Name, smbroker4.Name}
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []*platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
					reconcileUpdateCalledFor:  []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("When broker is registered in the platform and SM, but not yet taken over, and takeover is disabled it should not be taken over", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
				stubPlatformUpdateBroker(platformBrokerProxy)
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformbrokerNonProxy,
					platformbrokerNonProxy2,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker3,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: false,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []*platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
					reconcileUpdateCalledFor:  []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("When all brokers are registered in the platform and SM, but not yet taken over, and takeover is disabled they should not be taken over", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
				stubPlatformUpdateBroker(platformBrokerProxy)
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					platformbrokerNonProxy,
					platformbrokerNonProxy2,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker3,
					smbroker4,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: false,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []*platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{},
					reconcileUpdateCalledFor:  []*platform.ServiceBroker{},
				}
			},
		}),

		Entry("when a broker is renamed in the platform it should rename it back", testCase{
			// smBroker is registered in SM (as sm-smBroker-<id> in the platform), but it was renamed in the platform
			stubs: func() {
				stubPlatformOpsToSucceed()
				stubPlatformUpdateBroker(platformBrokerProxy)
			},
			platformBrokers: func() ([]*platform.ServiceBroker, error) {
				return []*platform.ServiceBroker{
					{
						Name:      brokerProxyName("smBroker1", smbroker2.ID), // the name of smBroker1 is changed in the platform
						BrokerURL: platformbroker1.BrokerURL,
						GUID:      platformbroker1.GUID,
					},
					platformbroker2,
				}, nil
			},
			smBrokers: func() ([]*types.ServiceBroker, error) {
				return []*types.ServiceBroker{
					smbroker1,
					smbroker2,
				}, nil
			},
			brokerBlacklist: func() []string {
				return []string{}
			},
			takeoverEnabled: true,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []*platform.ServiceBroker{},
					reconcileUpdateCalledFor: []*platform.ServiceBroker{
						{
							Name:      brokerProxyName("smBroker1", smbroker1.ID), // the broker should be updated with the name of smBroker1
							BrokerURL: platformbroker1.BrokerURL,
							GUID:      platformbroker1.GUID,
						},
					},
					reconcileDeleteCalledFor: []*platform.ServiceBroker{},
					reconcileCatalogCalledFor: []*platform.ServiceBroker{
						platformbroker2,
					},
				}
			},
		}),
	}

	DescribeTable("resync", func(t testCase) {
		smBrokers, err1 := t.smBrokers()
		platformBrokers, err2 := t.platformBrokers()

		fakeSMClient.GetBrokersReturns(smBrokers, err1)
		fakePlatformBrokerClient.GetBrokersReturns(platformBrokers, err2)
		t.stubs()

		reconcileSettings.BrokerBlacklist = t.brokerBlacklist()
		reconcileSettings.TakeoverEnabled = t.takeoverEnabled
		reconciler.Resyncer.Resync(context.TODO())

		invocations := append([]map[string][][]interface{}{}, fakeSMClient.Invocations(), fakePlatformCatalogFetcher.Invocations(), fakePlatformBrokerClient.Invocations())
		verifyInvocationsUseSameCorrelationID(invocations)

		if err1 != nil {
			Expect(len(fakePlatformBrokerClient.Invocations())).To(Equal(0))
			Expect(len(fakePlatformCatalogFetcher.Invocations())).To(Equal(0))
			Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))
			return
		}

		if err2 != nil {
			Expect(len(fakePlatformBrokerClient.Invocations())).To(Equal(1))
			Expect(len(fakePlatformCatalogFetcher.Invocations())).To(Equal(0))
			Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))
			return
		}

		Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))
		Expect(fakePlatformBrokerClient.GetBrokersCallCount()).To(Equal(1))

		expected := t.expectations()
		Expect(fakePlatformBrokerClient.CreateBrokerCallCount()).To(Equal(len(expected.reconcileCreateCalledFor)))
		for index, broker := range expected.reconcileCreateCalledFor {
			_, request := fakePlatformBrokerClient.CreateBrokerArgsForCall(index)
			Expect(request).To(Equal(&platform.CreateServiceBrokerRequest{
				Name:      broker.Name,
				BrokerURL: broker.BrokerURL,
			}))
		}

		Expect(fakePlatformCatalogFetcher.FetchCallCount()).To(Equal(len(expected.reconcileCatalogCalledFor)))
		for index, broker := range expected.reconcileCatalogCalledFor {
			_, serviceBroker := fakePlatformCatalogFetcher.FetchArgsForCall(index)
			Expect(serviceBroker).To(Equal(broker))
		}

		Expect(fakePlatformBrokerClient.DeleteBrokerCallCount()).To(Equal(len(expected.reconcileDeleteCalledFor)))
		for index, broker := range expected.reconcileDeleteCalledFor {
			_, request := fakePlatformBrokerClient.DeleteBrokerArgsForCall(index)
			Expect(request).To(Equal(&platform.DeleteServiceBrokerRequest{
				GUID: broker.GUID,
				Name: broker.Name,
			}))
		}

		Expect(fakePlatformBrokerClient.UpdateBrokerCallCount()).To(Equal(len(expected.reconcileUpdateCalledFor)))
		for index, broker := range expected.reconcileUpdateCalledFor {
			_, request := fakePlatformBrokerClient.UpdateBrokerArgsForCall(index)
			Expect(request).To(Equal(&platform.UpdateServiceBrokerRequest{
				GUID:      broker.GUID,
				Name:      broker.Name,
				BrokerURL: broker.BrokerURL,
			}))
		}

	}, entries...)

})
