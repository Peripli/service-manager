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

package sm

import (
	"testing"

	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/security/securityfakes"
	"github.com/Peripli/service-manager/storage"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSecurity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authentication Tests Suite")
}

type fakeSecureStorage struct {
	fetcher *securityfakes.FakeKeyFetcher
	setter  *securityfakes.FakeKeySetter
}

func (f *fakeSecureStorage) Fetcher() security.KeyFetcher {
	return f.fetcher
}

func (f *fakeSecureStorage) Setter() security.KeySetter {
	return f.setter
}

var _ = Describe("Initialize Secure Storage", func() {

	var keySetter *securityfakes.FakeKeySetter
	var keyFetcher *securityfakes.FakeKeyFetcher

	var secureStorage storage.Security

	BeforeEach(func() {
		keySetter = &securityfakes.FakeKeySetter{}
		keyFetcher = &securityfakes.FakeKeyFetcher{}
		secureStorage = &fakeSecureStorage{keyFetcher, keySetter}
	})

	Context("When node is leader", func() {
		It("Should generate encryption key and not wait", func() {
			keyFetcher.GetEncryptionKeyReturns([]byte{}, nil)
			keySetter.SetEncryptionKeyReturns(nil)
			err := initializeSecureStorage(secureStorage, true)
			Expect(keyFetcher.GetEncryptionKeyCallCount()).To(Equal(1))
			Expect(keySetter.SetEncryptionKeyCallCount()).To(Equal(1))
			Expect(err).To(BeNil())
		})
	})

	Context("When node is not leader", func() {
		Context("When leader generates key", func() {
			It("Should get encryption key from leader", func() {
				i := 0
				keyFetcher.GetEncryptionKeyStub = func() ([]byte, error) {
					if i == 0 {
						i++
						return []byte{}, nil
					}
					return []byte{1,2,3,4,5,6}, nil
				}
				err := initializeSecureStorage(secureStorage, false)
				Expect(keyFetcher.GetEncryptionKeyCallCount()).To(Equal(2))
				Expect(err).To(BeNil())
			})
		})
		Context("When leader failed to generate encryption key", func() {
			It("Returns error", func() {
				err := initializeSecureStorage(secureStorage, false)
				Expect(keyFetcher.GetEncryptionKeyCallCount()).To(Equal(getEncryptionKeyRetryCount + 1))
				Expect(err).ToNot(BeNil())
			})
		})
	})
})
