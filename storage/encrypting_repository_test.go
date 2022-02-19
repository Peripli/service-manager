package storage_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/security/securityfakes"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/storagefakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Encrypting Repository", func() {
	var fakeEncrypter *securityfakes.FakeEncrypter
	var fakeRepository *storagefakes.FakeStorage

	var repository *storage.TransactionalEncryptingRepository
	var err error

	var ctx context.Context
	var objWithDecryptedPassword types.Object
	var objWithEncryptedPassword types.Object

	var encryptCallsCountBeforeOp int
	var decryptCallsCountBeforeOp int

	BeforeEach(func() {
		ctx = context.TODO()

		objWithDecryptedPassword = &types.ServiceBroker{
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
		}

		objWithEncryptedPassword = &types.ServiceBroker{
			Base: types.Base{
				ID:    "id",
				Ready: true,
			},
			Credentials: &types.Credentials{
				Basic: &types.Basic{
					Username: "admin",
					Password: "encrypt" + objWithDecryptedPassword.(*types.ServiceBroker).Credentials.Basic.Password,
				},
			},
		}

		fakeEncrypter = &securityfakes.FakeEncrypter{}
		fakeEncrypter.EncryptCalls(func(ctx context.Context, plainText []byte, key []byte) ([]byte, error) {
			return append([]byte("encrypt"), plainText...), nil
		})

		fakeEncrypter.DecryptCalls(func(ctx context.Context, encryptedText []byte, key []byte) ([]byte, error) {
			encryptedString := string(encryptedText)
			if !strings.HasPrefix(encryptedString, "encrypt") {
				panic("decryption expects encrypted text")
			}
			decryptedText := strings.TrimPrefix(encryptedString, "encrypt")
			return []byte(decryptedText), nil
		})

		fakeRepository = &storagefakes.FakeStorage{}

		fakeRepository.CreateReturns(objWithEncryptedPassword, nil)

		fakeRepository.UpdateReturns(objWithEncryptedPassword, nil)

		fakeRepository.ListReturns(&types.ServiceBrokers{
			ServiceBrokers: []*types.ServiceBroker{
				objWithEncryptedPassword.(*types.ServiceBroker),
			},
		}, nil)

		fakeRepository.ListNoLabelsReturns(&types.ServiceBrokers{
			ServiceBrokers: []*types.ServiceBroker{
				objWithEncryptedPassword.(*types.ServiceBroker),
			},
		}, nil)

		fakeRepository.GetReturns(objWithEncryptedPassword.(*types.ServiceBroker), nil)

		fakeRepository.DeleteReturningReturns(&types.ServiceBrokers{
			ServiceBrokers: []*types.ServiceBroker{
				objWithEncryptedPassword.(*types.ServiceBroker),
			},
		}, nil)

		repository, err = storage.NewEncryptingRepository(fakeRepository, fakeEncrypter, []byte{})
		Expect(err).ToNot(HaveOccurred())

		encryptCallsCountBeforeOp = fakeEncrypter.EncryptCallCount()
		decryptCallsCountBeforeOp = fakeEncrypter.DecryptCallCount()
	})

	Describe("Create", func() {
		Context("when encrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.EncryptReturns(nil, fmt.Errorf("error"))

				_, err = repository.Create(ctx, objWithDecryptedPassword)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when decrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.DecryptReturns(nil, fmt.Errorf("error"))

				_, err = repository.Create(ctx, objWithDecryptedPassword)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeRepository.CreateReturns(nil, fmt.Errorf("error"))

				_, err = repository.Create(ctx, objWithDecryptedPassword)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no errors occur", func() {
			var returnedObj types.Object
			var err error
			var delegateCreateCallsCountBeforeOp int

			BeforeEach(func() {
				delegateCreateCallsCountBeforeOp = fakeRepository.CreateCallCount()
				returnedObj, err = repository.Create(ctx, objWithDecryptedPassword)
				Expect(err).ToNot(HaveOccurred())
			})

			It("encrypts the credentials once", func() {
				Expect(fakeEncrypter.EncryptCallCount() - encryptCallsCountBeforeOp).To(Equal(1))
			})

			It("invokes the delegate repository with object with encrypted credentials", func() {
				Expect(fakeRepository.CreateCallCount() - delegateCreateCallsCountBeforeOp).To(Equal(1))
				_, objectArg := fakeRepository.CreateArgsForCall(0)
				isPassEncrypted := strings.HasPrefix(objectArg.(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")
				Expect(isPassEncrypted).To(BeTrue())
			})

			It("decrypts the credentials once", func() {
				Expect(fakeEncrypter.DecryptCallCount() - decryptCallsCountBeforeOp).To(Equal(1))
			})

			It("returns an object with decrypted credentials", func() {
				isPassEncrypted := strings.HasPrefix(returnedObj.(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")
				Expect(isPassEncrypted).To(BeFalse())
			})
		})
	})

	Describe("List", func() {
		Context("when decrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.DecryptReturns(nil, fmt.Errorf("error"))

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

		Context("when no errors occur", func() {
			var returnedObjList types.ObjectList
			var err error

			BeforeEach(func() {
				returnedObjList, err = repository.List(ctx, types.ServiceBrokerType)
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not encrypt the credentials", func() {
				Expect(fakeEncrypter.EncryptCallCount() - encryptCallsCountBeforeOp).To(Equal(0))
			})

			It("decrypts the credentials once", func() {
				Expect(fakeEncrypter.DecryptCallCount() - decryptCallsCountBeforeOp).To(Equal(1))
			})

			It("returns an object with decrypted credentials", func() {
				for i := 0; i < returnedObjList.Len(); i++ {
					isPassEncrypted := strings.HasPrefix(returnedObjList.ItemAt(i).(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")
					Expect(isPassEncrypted).To(BeFalse())
				}
			})
		})
	})

	Describe("ListNoLabels", func() {
		Context("when decrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.DecryptReturns(nil, fmt.Errorf("error"))

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

		Context("when no errors occur", func() {
			var returnedObjList types.ObjectList
			var err error

			BeforeEach(func() {
				returnedObjList, err = repository.ListNoLabels(ctx, types.ServiceBrokerType)
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not encrypt the credentials", func() {
				Expect(fakeEncrypter.EncryptCallCount() - encryptCallsCountBeforeOp).To(Equal(0))
			})

			It("decrypts the credentials once", func() {
				Expect(fakeEncrypter.DecryptCallCount() - decryptCallsCountBeforeOp).To(Equal(1))
			})

			It("returns an object with decrypted credentials", func() {
				for i := 0; i < returnedObjList.Len(); i++ {
					isPassEncrypted := strings.HasPrefix(returnedObjList.ItemAt(i).(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")
					Expect(isPassEncrypted).To(BeFalse())
				}
			})
		})
	})

	Describe("Get", func() {
		Context("when decrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.DecryptReturns(nil, fmt.Errorf("error"))

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

		Context("when no errors occur", func() {
			var returnedObj types.Object
			var err error

			BeforeEach(func() {
				byID := query.ByField(query.EqualsOperator, "id", "id")
				returnedObj, err = repository.Get(ctx, types.ServiceBrokerType, byID)
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not encrypt the credentials", func() {
				Expect(fakeEncrypter.EncryptCallCount() - encryptCallsCountBeforeOp).To(Equal(0))
			})

			It("decrypts the credentials once", func() {
				Expect(fakeEncrypter.DecryptCallCount() - decryptCallsCountBeforeOp).To(Equal(1))
			})

			It("returns an object with decrypted credentials", func() {
				isPassEncrypted := strings.HasPrefix(returnedObj.(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")
				Expect(isPassEncrypted).To(BeFalse())
			})
		})
	})

	Describe("Update", func() {
		Context("when encrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.EncryptReturns(nil, fmt.Errorf("error"))

				_, err = repository.Update(ctx, objWithDecryptedPassword, types.LabelChanges{})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when decrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.DecryptReturns(nil, fmt.Errorf("error"))

				_, err = repository.Update(ctx, objWithDecryptedPassword, types.LabelChanges{})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeRepository.UpdateReturns(nil, fmt.Errorf("error"))

				_, err = repository.Update(ctx, objWithDecryptedPassword, types.LabelChanges{})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no errors occur", func() {
			var returnedObj types.Object
			var err error
			var delegateUpdateCallsCountBeforeOp int

			BeforeEach(func() {
				delegateUpdateCallsCountBeforeOp = fakeRepository.UpdateCallCount()
				returnedObj, err = repository.Update(ctx, objWithDecryptedPassword, types.LabelChanges{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("encrypts the credentials once", func() {
				Expect(fakeEncrypter.EncryptCallCount() - encryptCallsCountBeforeOp).To(Equal(1))
			})

			It("invokes the delegate repository with object with encrypted credentials", func() {
				Expect(fakeRepository.UpdateCallCount() - delegateUpdateCallsCountBeforeOp).To(Equal(1))
				_, objectArg, _, _ := fakeRepository.UpdateArgsForCall(0)
				isPassEncrypted := strings.HasPrefix(objectArg.(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")
				Expect(isPassEncrypted).To(BeTrue())
			})

			It("decrypts the credentials once", func() {
				Expect(fakeEncrypter.DecryptCallCount() - decryptCallsCountBeforeOp).To(Equal(1))
			})

			It("returns an object with decrypted credentials", func() {
				isPassEncrypted := strings.HasPrefix(returnedObj.(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")
				Expect(isPassEncrypted).To(BeFalse())
			})
		})
	})

	Describe("UpdateLabels", func() {
		It("does not invoke encryption or decryption and invokes the next in chain", func() {
			delegateUpdateCallsCountBeforeOp := fakeRepository.UpdateLabelsCallCount()
			err := repository.UpdateLabels(ctx, objWithDecryptedPassword.GetType(), objWithDecryptedPassword.GetID(), types.LabelChanges{})
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeEncrypter.EncryptCallCount() - encryptCallsCountBeforeOp).To(Equal(0))
			Expect(fakeEncrypter.DecryptCallCount() - decryptCallsCountBeforeOp).To(Equal(0))
			Expect(fakeRepository.UpdateLabelsCallCount() - delegateUpdateCallsCountBeforeOp).To(Equal(1))
		})
	})

	Describe("DeleteReturning", func() {
		Context("when decrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.DecryptReturns(nil, fmt.Errorf("error"))

				_, err = repository.DeleteReturning(ctx, types.ServiceBrokerType)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeRepository.DeleteReturns(fmt.Errorf("error"))

				err = repository.Delete(ctx, types.ServiceBrokerType)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no errors occur", func() {
			var returnedObjList types.ObjectList
			var err error

			BeforeEach(func() {
				returnedObjList, err = repository.DeleteReturning(ctx, types.ServiceBrokerType)
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not encrypt the credentials", func() {
				Expect(fakeEncrypter.EncryptCallCount() - encryptCallsCountBeforeOp).To(Equal(0))
			})

			It("decrypts the credentials once", func() {
				Expect(fakeEncrypter.DecryptCallCount() - decryptCallsCountBeforeOp).To(Equal(1))
			})

			It("returns an object with decrypted credentials", func() {
				for i := 0; i < returnedObjList.Len(); i++ {
					isPassEncrypted := strings.HasPrefix(returnedObjList.ItemAt(i).(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")
					Expect(isPassEncrypted).To(BeFalse())
				}
			})
		})
	})

	Describe("In transaction", func() {
		Context("when resource is created/updated/deleted/listed in transaction", func() {
			It("triggers encryption/decryption", func() {
				err := repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
					// verify create
					delegateCreateCallsCountBeforeOp := fakeRepository.CreateCallCount()
					returnedObj, err := repository.Create(ctx, objWithDecryptedPassword)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeRepository.CreateCallCount() - delegateCreateCallsCountBeforeOp).To(Equal(1))
					Expect(strings.HasPrefix(returnedObj.(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")).To(BeFalse())
					_, objectArg := fakeRepository.CreateArgsForCall(0)
					Expect(strings.HasPrefix(objectArg.(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")).To(BeTrue())

					// verify list
					delegateListCallsCountBeforeOp := fakeRepository.ListCallCount()
					returnedObjList, err := repository.List(ctx, types.ServiceBrokerType)
					Expect(err).To(HaveOccurred())
					Expect(fakeRepository.ListCallCount() - delegateListCallsCountBeforeOp).To(Equal(1))
					for i := 0; i < returnedObjList.Len(); i++ {
						isPassEncrypted := strings.HasPrefix(returnedObjList.ItemAt(i).(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")
						Expect(isPassEncrypted).To(BeFalse())
					}

					// verify update
					delegateUpdateCallsCountBeforeOp := fakeRepository.UpdateCallCount()
					returnedObj, err = repository.Update(ctx, objWithDecryptedPassword, types.LabelChanges{})
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeRepository.UpdateCallCount() - delegateUpdateCallsCountBeforeOp).To(Equal(1))
					_, objectArg, _, _ = fakeRepository.UpdateArgsForCall(0)
					Expect(strings.HasPrefix(objectArg.(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")).To(BeTrue())
					Expect(strings.HasPrefix(returnedObj.(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")).To(BeFalse())

					// verify get
					delegateGetCallsCountBeforeOp := fakeRepository.GetCallCount()
					byID := query.ByField(query.EqualsOperator, "id", "id")
					returnedObj, err = repository.Get(ctx, types.ServiceBrokerType, byID)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeRepository.GetCallCount() - delegateGetCallsCountBeforeOp).To(Equal(1))
					Expect(strings.HasPrefix(returnedObj.(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")).To(BeFalse())

					// verify delete
					delegateDeleteCallsCountBeforeOp := fakeRepository.DeleteCallCount()
					returnedObjList, err = repository.DeleteReturning(ctx, types.ServiceBrokerType)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeRepository.DeleteCallCount() - delegateDeleteCallsCountBeforeOp).To(Equal(1))
					for i := 0; i < returnedObjList.Len(); i++ {
						isPassEncrypted := strings.HasPrefix(returnedObjList.ItemAt(i).(*types.ServiceBroker).Credentials.Basic.Password, "encrypt")
						Expect(isPassEncrypted).To(BeFalse())
					}

					return nil
				})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
