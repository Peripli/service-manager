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

package reconcile

import (
	"context"
	"fmt"
	"sync"

	"github.com/Peripli/service-manager/pkg/sbproxy/platform"
	"github.com/Peripli/service-manager/pkg/sbproxy/platform/platformfakes"
	"github.com/Peripli/service-manager/pkg/sbproxy/sm"
	"github.com/Peripli/service-manager/pkg/sbproxy/sm/smfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

var _ = Describe("ReconcilationTask", func() {
	const fakeAppHost = "https://smproxy.com"

	var (
		fakeSMClient *smfakes.FakeClient

		fakePlatformCatalogFetcher *platformfakes.FakeCatalogFetcher
		fakePlatformServiceAccess  *platformfakes.FakeServiceAccess
		fakePlatformBrokerClient   *platformfakes.FakeClient

		waitGroup *sync.WaitGroup

		reconcilationTask *ReconcilationTask

		smbroker1 sm.Broker
		smbroker2 sm.Broker

		platformbroker1        platform.ServiceBroker
		platformbroker2        platform.ServiceBroker
		platformbrokerNonProxy platform.ServiceBroker
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
		fakePlatformServiceAccess.EnableAccessForServiceReturns(nil)
		fakePlatformCatalogFetcher.FetchReturns(nil)
	}

	BeforeEach(func() {
		fakeSMClient = &smfakes.FakeClient{}
		fakePlatformBrokerClient = &platformfakes.FakeClient{}
		fakePlatformCatalogFetcher = &platformfakes.FakeCatalogFetcher{}
		fakePlatformServiceAccess = &platformfakes.FakeServiceAccess{}

		waitGroup = &sync.WaitGroup{}

		reconcilationTask = NewTask(context.TODO(), waitGroup, struct {
			*platformfakes.FakeCatalogFetcher
			*platformfakes.FakeServiceAccess
			*platformfakes.FakeClient
		}{
			FakeCatalogFetcher: fakePlatformCatalogFetcher,
			FakeServiceAccess:  fakePlatformServiceAccess,
			FakeClient:         fakePlatformBrokerClient,
		}, fakeSMClient, fakeAppHost)

		smbroker1 = sm.Broker{
			ID:        "smBrokerID1",
			BrokerURL: "https://smBroker1.com",
			Catalog: &osbc.CatalogResponse{
				Services: []osbc.Service{
					{
						ID:                  "smBroker1ServiceID1",
						Name:                "smBroker1Service1",
						Description:         "description",
						Bindable:            true,
						BindingsRetrievable: true,
						Plans: []osbc.Plan{
							{
								ID:          "smBroker1ServiceID1PlanID1",
								Name:        "smBroker1Service1Plan1",
								Description: "description",
							},
							{
								ID:          "smBroker1ServiceID1PlanID2",
								Name:        "smBroker1Service1Plan2",
								Description: "description",
							},
						},
					},
				},
			},
		}

		smbroker2 = sm.Broker{
			ID:        "smBrokerID2",
			BrokerURL: "https://smBroker2.com",
			Catalog: &osbc.CatalogResponse{
				Services: []osbc.Service{
					{
						ID:                  "smBroker2ServiceID1",
						Name:                "smBroker2Service1",
						Description:         "description",
						Bindable:            true,
						BindingsRetrievable: true,
						Plans: []osbc.Plan{
							{
								ID:          "smBroker2ServiceID1PlanID1",
								Name:        "smBroker2Service1Plan1",
								Description: "description",
							},
							{
								ID:          "smBroker2ServiceID1PlanID2",
								Name:        "smBroker2Service1Plan2",
								Description: "description",
							},
						},
					},
				},
			},
		}

		platformbroker1 = platform.ServiceBroker{
			GUID:      "platformBrokerID1",
			Name:      ProxyBrokerPrefix + "smBrokerID1",
			BrokerURL: fakeAppHost + "/" + smbroker1.ID,
		}

		platformbroker2 = platform.ServiceBroker{
			GUID:      "platformBrokerID2",
			Name:      ProxyBrokerPrefix + "smBrokerID2",
			BrokerURL: fakeAppHost + "/" + smbroker2.ID,
		}

		platformbrokerNonProxy = platform.ServiceBroker{
			GUID:      "platformBrokerID3",
			Name:      "platformBroker3",
			BrokerURL: "https://platformBroker3.com",
		}
	})

	type expectations struct {
		reconcileCreateCalledFor  []platform.ServiceBroker
		reconcileDeleteCalledFor  []platform.ServiceBroker
		reconcileCatalogCalledFor []platform.ServiceBroker
		reconcileAccessCalledFor  []osbc.CatalogResponse
	}

	type testCase struct {
		stubs           func()
		platformBrokers func() ([]platform.ServiceBroker, error)
		smBrokers       func() ([]sm.Broker, error)

		expectations func() expectations
	}

	entries := []TableEntry{
		Entry("When fetching brokers from SM fails no reconcilation should be done", testCase{
			stubs: func() {

			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return nil, fmt.Errorf("error fetching brokers")
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor:  []osbc.CatalogResponse{},
				}
			},
		}),

		Entry("When fetching brokers from platform fails no reconcilation should be done", testCase{
			stubs: func() {

			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return nil, fmt.Errorf("error fetching brokers")
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor:  []osbc.CatalogResponse{},
				}
			},
		}),

		Entry("When platform broker op fails reconcilation continues with the next broker", testCase{
			stubs: func() {
				fakePlatformBrokerClient.DeleteBrokerReturns(fmt.Errorf("error"))
				fakePlatformServiceAccess.EnableAccessForServiceReturns(fmt.Errorf("error"))
				fakePlatformCatalogFetcher.FetchReturns(fmt.Errorf("error"))
				fakePlatformBrokerClient.CreateBrokerStub = stubCreateBrokerToReturnError
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker2,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
				}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileDeleteCalledFor: []platform.ServiceBroker{
						platformbroker2,
					},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor:  []osbc.CatalogResponse{},
				}
			},
		}),

		Entry("When broker from SM has no catalog reconcilation continues with the next broker", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
					platformbroker2,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				smbroker1.Catalog = nil
				return []sm.Broker{
					smbroker1,
					smbroker2,
				}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{
						platformbroker1,
						platformbroker2,
					},
					reconcileAccessCalledFor: []osbc.CatalogResponse{
						*smbroker2.Catalog,
					},
				}
			},
		}),

		Entry("When broker is in SM and is missing from platform it should be created and access enabled", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
					smbroker2,
				}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{
						platformbroker1,
						platformbroker2,
					},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor: []osbc.CatalogResponse{
						*smbroker1.Catalog,
						*smbroker2.Catalog,
					},
				}
			},
		}),

		Entry("When broker is in SM and is also in platform it should be catalog refetched and access enabled", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
				}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileAccessCalledFor: []osbc.CatalogResponse{
						*smbroker1.Catalog,
					},
				}
			},
		}),

		Entry("When broker is missing from SM but is in platform it should be deleted", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor:  []osbc.CatalogResponse{},
				}
			},
		}),

		Entry("When broker is missing from SM but is in platform that is not represented by the proxy should be ignored", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbrokerNonProxy,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor:  []osbc.CatalogResponse{},
				}
			},
		}),
	}

	DescribeTable("Run", func(t testCase) {
		smBrokers, err1 := t.smBrokers()
		platformBrokers, err2 := t.platformBrokers()

		fakeSMClient.GetBrokersReturns(smBrokers, err1)
		fakePlatformBrokerClient.GetBrokersReturns(platformBrokers, err2)
		t.stubs()

		reconcilationTask.Run()

		if err1 != nil {
			Expect(len(fakePlatformBrokerClient.Invocations())).To(Equal(1))
			Expect(len(fakePlatformCatalogFetcher.Invocations())).To(Equal(0))
			Expect(len(fakePlatformServiceAccess.Invocations())).To(Equal(0))
			Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))
			return
		}

		if err2 != nil {
			Expect(len(fakePlatformBrokerClient.Invocations())).To(Equal(1))
			Expect(len(fakePlatformCatalogFetcher.Invocations())).To(Equal(0))
			Expect(len(fakePlatformServiceAccess.Invocations())).To(Equal(0))
			Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(0))
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
			Expect(serviceBroker).To(Equal(&broker))
		}

		servicesCount := 0
		index := 0
		for _, catalog := range expected.reconcileAccessCalledFor {
			for _, service := range catalog.Services {
				_, _, serviceID := fakePlatformServiceAccess.EnableAccessForServiceArgsForCall(index)
				Expect(serviceID).To(Equal(service.ID))
				servicesCount++
				index++
			}
		}
		Expect(fakePlatformServiceAccess.EnableAccessForServiceCallCount()).To(Equal(servicesCount))

		Expect(fakePlatformBrokerClient.DeleteBrokerCallCount()).To(Equal(len(expected.reconcileDeleteCalledFor)))
		for index, broker := range expected.reconcileDeleteCalledFor {
			_, request := fakePlatformBrokerClient.DeleteBrokerArgsForCall(index)
			Expect(request).To(Equal(&platform.DeleteServiceBrokerRequest{
				GUID: broker.GUID,
				Name: broker.Name,
			}))
		}
	}, entries...)

	Describe("Settings", func() {
		Describe("Validate", func() {
			Context("when host is missing", func() {
				It("returns an error", func() {
					settings := DefaultSettings()
					err := settings.Validate()

					Expect(err).Should(HaveOccurred())
				})
			})
		})
	})
})
