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

package catalog_test

import (
	"context"
	"fmt"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage/catalog"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage/storagefakes"
)

var _ = Describe("Catalog Load", func() {
	ctx := context.TODO()
	brokerID := "brokerID"

	var repository *storagefakes.FakeStorage

	BeforeEach(func() {
		repository = &storagefakes.FakeStorage{}
	})

	Context("When list service offerings returns error", func() {
		expectedError := fmt.Errorf("error loading service offerings")
		BeforeEach(func() {
			repository.ListReturns(nil, expectedError)
		})
		It("Returns error", func() {
			offerings, err := catalog.Load(ctx, brokerID, repository)
			Expect(offerings).To(BeNil())
			Expect(err).To(Equal(expectedError))
		})
	})

	Context("When service offerings are loaded from the storage", func() {
		offeringsList := &types.ServiceOfferings{
			ServiceOfferings: []*types.ServiceOffering{
				{
					Base: types.Base{
						ID: "service-offering-id",
					},
				},
			},
		}
		Context("When no service offerings are found", func() {
			BeforeEach(func() {
				repository.ListReturns(&types.ServiceOfferings{}, nil)
			})
			It("Returns empty list", func() {
				offerings, err := catalog.Load(ctx, brokerID, repository)
				Expect(offerings).ToNot(BeNil())
				Expect(err).To(BeNil())
			})
		})
		Context("When list service plans returns error", func() {
			expectedError := fmt.Errorf("error loading service plans")
			BeforeEach(func() {
				repository.ListStub = func(ctx context.Context, objectType types.ObjectType, criterion ...query.Criterion) (types.ObjectList, error) {
					if objectType == types.ServiceOfferingType {
						return offeringsList, nil
					}
					return nil, expectedError
				}
			})
			It("Returns error", func() {
				offerings, err := catalog.Load(ctx, brokerID, repository)
				Expect(offerings).To(BeNil())
				Expect(err).To(Equal(expectedError))
			})
		})
		Context("When list service plans loads plans", func() {
			plansList := &types.ServicePlans{
				ServicePlans: []*types.ServicePlan{
					{
						Base: types.Base{
							ID: "service-plan-id",
						},
					},
				},
			}
			BeforeEach(func() {
				repository.ListStub = func(ctx context.Context, objectType types.ObjectType, criterion ...query.Criterion) (types.ObjectList, error) {
					if objectType == types.ServiceOfferingType {
						return offeringsList, nil
					}
					return plansList, nil
				}
			})
			It("Returns result", func() {
				offerings, err := catalog.Load(ctx, brokerID, repository)
				expectedOffering := offeringsList.ServiceOfferings[0]
				expectedOffering.Plans = []*types.ServicePlan{plansList.ServicePlans[0]}
				Expect(offerings.ServiceOfferings).To(ConsistOf(expectedOffering))
				Expect(err).To(BeNil())
			})
		})
	})
})
