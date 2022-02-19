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

package storage_test

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/security"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/security/securityfakes"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/storagefakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Integrity Repository", func() {
	var fakeIntegrityProcessor *securityfakes.FakeIntegrityProcessor
	var fakeRepository *storagefakes.FakeStorage

	var repository *storage.TransactionalIntegrityRepository
	var err error

	var ctx context.Context
	var object types.Object

	var validateCallsCountBeforeOp int
	var calculateCallsCountBeforeOp int
	var randomIntegrity []byte

	BeforeEach(func() {
		ctx = context.TODO()

		object = &types.ServiceBroker{
			Base: types.Base{
				ID:    "id",
				Ready: true,
			},
			Credentials: &types.Credentials{
				Basic: &types.Basic{
					Username: "admin",
					Password: "admin",
				},
			},
			BrokerURL: "http://example.com",
		}

		fakeRepository = &storagefakes.FakeStorage{}

		fakeRepository.CreateReturns(object, nil)

		fakeRepository.UpdateReturns(object, nil)

		fakeRepository.ListReturns(&types.ServiceBrokers{
			ServiceBrokers: []*types.ServiceBroker{
				object.(*types.ServiceBroker),
			},
		}, nil)

		fakeRepository.ListNoLabelsReturns(&types.ServiceBrokers{
			ServiceBrokers: []*types.ServiceBroker{
				object.(*types.ServiceBroker),
			},
		}, nil)

		fakeRepository.GetReturns(object.(*types.ServiceBroker), nil)

		fakeRepository.DeleteReturningReturns(&types.ServiceBrokers{
			ServiceBrokers: []*types.ServiceBroker{
				object.(*types.ServiceBroker),
			},
		}, nil)

		bytes := make([]byte, 32)
		rand.Read(bytes)
		copy(randomIntegrity[:], bytes[:])

		fakeIntegrityProcessor = &securityfakes.FakeIntegrityProcessor{}
		fakeIntegrityProcessor.CalculateIntegrityReturns(randomIntegrity, nil)
		fakeIntegrityProcessor.ValidateIntegrityReturns(true)

		repository = storage.NewIntegrityRepository(fakeRepository, fakeIntegrityProcessor)

		validateCallsCountBeforeOp = fakeIntegrityProcessor.ValidateIntegrityCallCount()
		calculateCallsCountBeforeOp = fakeIntegrityProcessor.CalculateIntegrityCallCount()

	})

	Describe("Create", func() {
		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeRepository.CreateReturns(nil, fmt.Errorf("error"))

				_, err = repository.Create(ctx, object)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when calculate integrity fails", func() {
			It("returns an error", func() {
				fakeIntegrityProcessor.CalculateIntegrityReturns([]byte{}, fmt.Errorf("error"))

				_, err = repository.Create(ctx, object)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no errors occur", func() {
			var err error
			var delegateCreateCallsCountBeforeOp int

			BeforeEach(func() {
				delegateCreateCallsCountBeforeOp = fakeRepository.CreateCallCount()
				_, err = repository.Create(ctx, object)
				fakeIntegrityProcessor.CalculateIntegrityReturns(randomIntegrity, nil)

				Expect(err).ToNot(HaveOccurred())
			})

			It("calculates integrity once", func() {
				Expect(fakeIntegrityProcessor.CalculateIntegrityCallCount() - validateCallsCountBeforeOp).To(Equal(1))
			})

			It("invokes the delegate repository with object with set integrity", func() {
				Expect(fakeRepository.CreateCallCount() - delegateCreateCallsCountBeforeOp).To(Equal(1))
				_, objectArg := fakeRepository.CreateArgsForCall(0)
				Expect(objectArg.(security.IntegralObject).GetIntegrity()).To(Equal(randomIntegrity))
			})
		})
	})

	Describe("List", func() {
		Context("when integrity is not valid", func() {
			It("returns an error", func() {
				fakeIntegrityProcessor.ValidateIntegrityReturns(false)

				_, err = repository.List(ctx, types.ServiceBrokerType)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeRepository.ListReturns(nil, fmt.Errorf("error"))

				_, err = repository.List(ctx, types.ServiceBrokerType)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when integrity is valid", func() {

			It("returns objects", func() {
				fakeIntegrityProcessor.ValidateIntegrityReturns(true)

				_, err := repository.List(ctx, types.ServiceBrokerType)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("ListNoLabels", func() {
		Context("when integrity is not valid", func() {
			It("returns an error", func() {
				fakeIntegrityProcessor.ValidateIntegrityReturns(false)

				_, err = repository.ListNoLabels(ctx, types.ServiceBrokerType)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeRepository.ListNoLabelsReturns(nil, fmt.Errorf("error"))

				_, err = repository.ListNoLabels(ctx, types.ServiceBrokerType)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when integrity is valid", func() {

			It("returns objects", func() {
				fakeIntegrityProcessor.ValidateIntegrityReturns(true)

				_, err := repository.ListNoLabels(ctx, types.ServiceBrokerType)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("Get", func() {
		Context("when integrity is not valid", func() {
			It("returns an error", func() {
				fakeIntegrityProcessor.ValidateIntegrityReturns(false)

				byID := query.ByField(query.EqualsOperator, "id", "id")
				_, err = repository.Get(ctx, types.ServiceBrokerType, byID)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeRepository.GetReturns(nil, fmt.Errorf("error"))

				byID := query.ByField(query.EqualsOperator, "id", "id")
				_, err = repository.Get(ctx, types.ServiceBrokerType, byID)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when integrity is valid ", func() {
			It("returns successfully", func() {
				fakeIntegrityProcessor.ValidateIntegrityReturns(true)

				byID := query.ByField(query.EqualsOperator, "id", "id")
				_, err = repository.Get(ctx, types.ServiceBrokerType, byID)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("Update", func() {
		Context("when calculate integrity fails", func() {
			It("returns an error", func() {
				fakeIntegrityProcessor.CalculateIntegrityReturns([]byte{}, fmt.Errorf("error"))

				_, err = repository.Update(ctx, object, types.LabelChanges{})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeRepository.UpdateReturns(nil, fmt.Errorf("error"))

				_, err = repository.Update(ctx, object, types.LabelChanges{})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when credentials are changed", func() {
			var newIntegrity [32]byte
			BeforeEach(func() {
				bytes := make([]byte, 32)
				rand.Read(bytes)
				copy(newIntegrity[:], bytes[:])
				fakeIntegrityProcessor.CalculateIntegrityReturns(newIntegrity[:], nil)
			})
			It("sets a new integrity", func() {
				broker := object.(*types.ServiceBroker)
				oldIntegrity := broker.Credentials.Integrity
				Expect(newIntegrity).ToNot(Equal(oldIntegrity))
				updatedObject, err := repository.Update(ctx, broker, types.LabelChanges{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedObject.(security.IntegralObject).GetIntegrity()).To(Equal(newIntegrity[:]))
				Expect(updatedObject.(security.IntegralObject).GetIntegrity()).ToNot(Equal(oldIntegrity))
			})
		})

		Context("when no errors occur", func() {
			var delegateUpdateCallsCountBeforeOp int

			BeforeEach(func() {
				delegateUpdateCallsCountBeforeOp = fakeRepository.UpdateCallCount()
				_, err := repository.Update(ctx, object, types.LabelChanges{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("calculates integrity once", func() {
				Expect(fakeIntegrityProcessor.CalculateIntegrityCallCount() - validateCallsCountBeforeOp).To(Equal(1))
			})

			It("invokes the delegate repository with object with set integrity", func() {
				Expect(fakeRepository.UpdateCallCount() - delegateUpdateCallsCountBeforeOp).To(Equal(1))
				_, objectArg, _, _ := fakeRepository.UpdateArgsForCall(0)
				Expect(objectArg.(security.IntegralObject).GetIntegrity()).To(Equal(randomIntegrity))
			})
		})
	})

	Describe("UpdateLabels", func() {
		It("does not invoke integrity calculation or validation and invokes the next in chain", func() {
			delegateUpdateCallsCountBeforeOp := fakeRepository.UpdateLabelsCallCount()
			err := repository.UpdateLabels(ctx, object.GetType(), object.GetID(), types.LabelChanges{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(fakeIntegrityProcessor.Invocations())).To(Equal(0))
			Expect(fakeRepository.UpdateLabelsCallCount() - delegateUpdateCallsCountBeforeOp).To(Equal(1))
		})
	})

	Describe("DeleteReturning", func() {
		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeRepository.DeleteReturns(fmt.Errorf("error"))

				err = repository.Delete(ctx, types.ServiceBrokerType)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no errors occur", func() {
			var err error

			BeforeEach(func() {
				_, err = repository.DeleteReturning(ctx, types.ServiceBrokerType)
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not validate the integrity", func() {
				Expect(fakeIntegrityProcessor.ValidateIntegrityCallCount() - validateCallsCountBeforeOp).To(Equal(0))
			})

			It("does not calculate the integrity", func() {
				Expect(fakeIntegrityProcessor.CalculateIntegrityCallCount() - calculateCallsCountBeforeOp).To(Equal(0))
			})
		})
	})

	Describe("In transaction", func() {
		Context("when resource is created/updated/deleted/listed in transaction", func() {
			It("triggers validation/calculation", func() {
				err := repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
					var err error

					// verify create
					delegateCreateCallsCountBeforeOp := fakeRepository.CreateCallCount()
					returnedObj, err := repository.Create(ctx, object)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeRepository.CreateCallCount() - delegateCreateCallsCountBeforeOp).To(Equal(1))
					Expect(returnedObj.(security.IntegralObject).GetIntegrity()).To(Equal(randomIntegrity))
					_, objectArg := fakeRepository.CreateArgsForCall(0)
					Expect(objectArg.(security.IntegralObject).GetIntegrity()).To(Equal(randomIntegrity))

					// verify list
					delegateListCallsCountBeforeOp := fakeRepository.ListCallCount()
					_, err = repository.List(ctx, types.ServiceBrokerType)
					Expect(err).To(HaveOccurred())
					Expect(fakeRepository.ListCallCount() - delegateListCallsCountBeforeOp).To(Equal(1))

					// verify update
					delegateUpdateCallsCountBeforeOp := fakeRepository.UpdateCallCount()
					returnedObj, err = repository.Update(ctx, object, types.LabelChanges{})
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeRepository.UpdateCallCount() - delegateUpdateCallsCountBeforeOp).To(Equal(1))
					_, objectArg, _, _ = fakeRepository.UpdateArgsForCall(0)
					Expect(returnedObj.(security.IntegralObject).GetIntegrity()).To(Equal(randomIntegrity))
					Expect(objectArg.(security.IntegralObject).GetIntegrity()).To(Equal(randomIntegrity))

					// verify update labels
					delegateUpdateLabelsCallsCountBeforeOp := fakeRepository.UpdateLabelsCallCount()
					err = repository.UpdateLabels(ctx, object.GetType(), object.GetID(), types.LabelChanges{})
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeRepository.UpdateLabelsCallCount() - delegateUpdateLabelsCallsCountBeforeOp).To(Equal(1))

					// verify get
					delegateGetCallsCountBeforeOp := fakeRepository.GetCallCount()
					byID := query.ByField(query.EqualsOperator, "id", "id")
					returnedObj, err = repository.Get(ctx, types.ServiceBrokerType, byID)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeRepository.GetCallCount() - delegateGetCallsCountBeforeOp).To(Equal(1))
					Expect(returnedObj.(security.IntegralObject).GetIntegrity()).To(Equal(randomIntegrity))

					// verify delete
					delegateDeleteCallsCountBeforeOp := fakeRepository.DeleteCallCount()
					_, err = repository.DeleteReturning(ctx, types.ServiceBrokerType)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeRepository.DeleteCallCount() - delegateDeleteCallsCountBeforeOp).To(Equal(1))

					return nil
				})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
