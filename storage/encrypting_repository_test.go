package storage_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/security/securityfakes"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/storagefakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Encrypting Repository", func() {
	var fakeEncrypter *securityfakes.FakeEncrypter
	var fakeSecuredRepository *storagefakes.FakeSecuredTransactionalRepository

	var repository *storage.EncryptingRepository
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
				ID: "id",
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
				ID: "id",
			},
			Credentials: &types.Credentials{
				Basic: &types.Basic{
					Username: "admin",
					Password: "encrypt" + objWithDecryptedPassword.(types.Secured).GetCredentials().Basic.Password,
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

		fakeSecuredRepository = &storagefakes.FakeSecuredTransactionalRepository{}

		fakeSecuredRepository.CreateReturns(objWithEncryptedPassword, nil)

		fakeSecuredRepository.UpdateReturns(objWithEncryptedPassword, nil)

		fakeSecuredRepository.ListReturns(&types.ServiceBrokers{
			ServiceBrokers: []*types.ServiceBroker{
				objWithEncryptedPassword.(*types.ServiceBroker),
			},
		}, nil)

		fakeSecuredRepository.GetReturns(objWithEncryptedPassword.(*types.ServiceBroker), nil)

		fakeSecuredRepository.DeleteReturns(&types.ServiceBrokers{
			ServiceBrokers: []*types.ServiceBroker{
				objWithEncryptedPassword.(*types.ServiceBroker),
			},
		}, nil)

		repository, err = storage.NewEncryptingRepository(ctx, fakeSecuredRepository, fakeEncrypter)
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
				fakeSecuredRepository.CreateReturns(nil, fmt.Errorf("error"))

				_, err = repository.Create(ctx, objWithDecryptedPassword)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no errors occur", func() {
			var returnedObj types.Object
			var err error
			var delegateCreateCallsCountBeforeOp int

			BeforeEach(func() {
				delegateCreateCallsCountBeforeOp = fakeSecuredRepository.CreateCallCount()
				returnedObj, err = repository.Create(ctx, objWithDecryptedPassword)
				Expect(err).ToNot(HaveOccurred())
			})

			It("encrypts the credentials once", func() {
				Expect(fakeEncrypter.EncryptCallCount() - encryptCallsCountBeforeOp).To(Equal(1))
			})

			It("invokes the delegate repository with object with encrypted credentials", func() {
				Expect(fakeSecuredRepository.CreateCallCount() - delegateCreateCallsCountBeforeOp).To(Equal(1))
				_, objectArg := fakeSecuredRepository.CreateArgsForCall(0)
				isPassEncrypted := strings.HasPrefix(objectArg.(types.Secured).GetCredentials().Basic.Password, "encrypt")
				Expect(isPassEncrypted).To(BeTrue())
			})

			It("decrypts the credentials once", func() {
				Expect(fakeEncrypter.DecryptCallCount() - decryptCallsCountBeforeOp).To(Equal(1))
			})

			It("returns an object with decrypted credentials", func() {
				isPassEncrypted := strings.HasPrefix(returnedObj.(types.Secured).GetCredentials().Basic.Password, "encrypt")
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
				fakeSecuredRepository.ListReturns(nil, fmt.Errorf("error"))

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
					isPassEncrypted := strings.HasPrefix(returnedObjList.ItemAt(i).(types.Secured).GetCredentials().Basic.Password, "encrypt")
					Expect(isPassEncrypted).To(BeFalse())
				}
			})
		})
	})

	Describe("Get", func() {
		Context("when decrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.DecryptReturns(nil, fmt.Errorf("error"))

				_, err = repository.Get(ctx, types.ServiceBrokerType, "id")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeSecuredRepository.GetReturns(nil, fmt.Errorf("error"))

				_, err = repository.Get(ctx, types.ServiceBrokerType, "id")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no errors occur", func() {
			var returnedObj types.Object
			var err error

			BeforeEach(func() {
				returnedObj, err = repository.Get(ctx, types.ServiceBrokerType, "id")
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not encrypt the credentials", func() {
				Expect(fakeEncrypter.EncryptCallCount() - encryptCallsCountBeforeOp).To(Equal(0))
			})

			It("decrypts the credentials once", func() {
				Expect(fakeEncrypter.DecryptCallCount() - decryptCallsCountBeforeOp).To(Equal(1))
			})

			It("returns an object with decrypted credentials", func() {
				isPassEncrypted := strings.HasPrefix(returnedObj.(types.Secured).GetCredentials().Basic.Password, "encrypt")
				Expect(isPassEncrypted).To(BeFalse())
			})
		})
	})

	Describe("Update", func() {
		Context("when encrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.EncryptReturns(nil, fmt.Errorf("error"))

				_, err = repository.Update(ctx, objWithDecryptedPassword)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when decrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.DecryptReturns(nil, fmt.Errorf("error"))

				_, err = repository.Update(ctx, objWithDecryptedPassword)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeSecuredRepository.UpdateReturns(nil, fmt.Errorf("error"))

				_, err = repository.Update(ctx, objWithDecryptedPassword)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no errors occur", func() {
			var returnedObj types.Object
			var err error
			var delegateUpdateCallsCountBeforeOp int

			BeforeEach(func() {
				delegateUpdateCallsCountBeforeOp = fakeSecuredRepository.UpdateCallCount()
				returnedObj, err = repository.Update(ctx, objWithDecryptedPassword)
				Expect(err).ToNot(HaveOccurred())
			})

			It("encrypts the credentials once", func() {
				Expect(fakeEncrypter.EncryptCallCount() - encryptCallsCountBeforeOp).To(Equal(1))
			})

			It("invokes the delegate repository with object with encrypted credentials", func() {
				Expect(fakeSecuredRepository.UpdateCallCount() - delegateUpdateCallsCountBeforeOp).To(Equal(1))
				_, objectArg, _ := fakeSecuredRepository.UpdateArgsForCall(0)
				isPassEncrypted := strings.HasPrefix(objectArg.(types.Secured).GetCredentials().Basic.Password, "encrypt")
				Expect(isPassEncrypted).To(BeTrue())
			})

			It("decrypts the credentials once", func() {
				Expect(fakeEncrypter.DecryptCallCount() - decryptCallsCountBeforeOp).To(Equal(1))
			})

			It("returns an object with decrypted credentials", func() {
				isPassEncrypted := strings.HasPrefix(returnedObj.(types.Secured).GetCredentials().Basic.Password, "encrypt")
				Expect(isPassEncrypted).To(BeFalse())
			})
		})
	})

	Describe("Delete", func() {
		Context("when decrypting fails", func() {
			It("returns an error", func() {
				fakeEncrypter.DecryptReturns(nil, fmt.Errorf("error"))

				_, err = repository.Delete(ctx, types.ServiceBrokerType)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when delegate call fails", func() {
			It("returns an error", func() {
				fakeSecuredRepository.DeleteReturns(nil, fmt.Errorf("error"))

				_, err = repository.Delete(ctx, types.ServiceBrokerType)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no errors occur", func() {
			var returnedObjList types.ObjectList
			var err error

			BeforeEach(func() {
				returnedObjList, err = repository.Delete(ctx, types.ServiceBrokerType)
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
					isPassEncrypted := strings.HasPrefix(returnedObjList.ItemAt(i).(types.Secured).GetCredentials().Basic.Password, "encrypt")
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
					delegateCreateCallsCountBeforeOp := fakeSecuredRepository.CreateCallCount()
					returnedObj, err := repository.Create(ctx, objWithDecryptedPassword)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeSecuredRepository.CreateCallCount() - delegateCreateCallsCountBeforeOp).To(Equal(1))
					Expect(strings.HasPrefix(returnedObj.(types.Secured).GetCredentials().Basic.Password, "encrypt")).To(BeFalse())
					_, objectArg := fakeSecuredRepository.CreateArgsForCall(0)
					Expect(strings.HasPrefix(objectArg.(types.Secured).GetCredentials().Basic.Password, "encrypt")).To(BeTrue())

					// verify list
					delegateListCallsCountBeforeOp := fakeSecuredRepository.ListCallCount()
					returnedObjList, err := repository.List(ctx, types.ServiceBrokerType)
					Expect(err).To(HaveOccurred())
					Expect(fakeSecuredRepository.ListCallCount() - delegateListCallsCountBeforeOp).To(Equal(1))
					for i := 0; i < returnedObjList.Len(); i++ {
						isPassEncrypted := strings.HasPrefix(returnedObjList.ItemAt(i).(types.Secured).GetCredentials().Basic.Password, "encrypt")
						Expect(isPassEncrypted).To(BeFalse())
					}

					// verify update
					delegateUpdateCallsCountBeforeOp := fakeSecuredRepository.UpdateCallCount()
					returnedObj, err = repository.Update(ctx, objWithDecryptedPassword)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeSecuredRepository.UpdateCallCount() - delegateUpdateCallsCountBeforeOp).To(Equal(1))
					_, objectArg, _ = fakeSecuredRepository.UpdateArgsForCall(0)
					Expect(strings.HasPrefix(objectArg.(types.Secured).GetCredentials().Basic.Password, "encrypt")).To(BeTrue())
					Expect(strings.HasPrefix(returnedObj.(types.Secured).GetCredentials().Basic.Password, "encrypt")).To(BeFalse())

					// verify get
					delegateGetCallsCountBeforeOp := fakeSecuredRepository.GetCallCount()
					returnedObj, err = repository.Get(ctx, types.ServiceBrokerType, "id")
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeSecuredRepository.GetCallCount() - delegateGetCallsCountBeforeOp).To(Equal(1))
					Expect(strings.HasPrefix(returnedObj.(types.Secured).GetCredentials().Basic.Password, "encrypt")).To(BeFalse())

					// verify delete
					delegateDeleteCallsCountBeforeOp := fakeSecuredRepository.DeleteCallCount()
					returnedObjList, err = repository.Delete(ctx, types.ServiceBrokerType)
					Expect(err).ToNot(HaveOccurred())
					Expect(fakeSecuredRepository.DeleteCallCount() - delegateDeleteCallsCountBeforeOp).To(Equal(1))
					for i := 0; i < returnedObjList.Len(); i++ {
						isPassEncrypted := strings.HasPrefix(returnedObjList.ItemAt(i).(types.Secured).GetCredentials().Basic.Password, "encrypt")
						Expect(isPassEncrypted).To(BeFalse())
					}

					return nil
				})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
