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

package security_test

import (
	"crypto/rand"
	"fmt"
	"log"
	"testing"

	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/security/securityfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Encrypter Test Suite")
}

var _ = Describe("Encrypter", func() {

	var encrypter security.Encrypter
	var fetcher *securityfakes.FakeKeyFetcher
	plaintext := []byte("plaintext")
	generateEncryptionKey := func() []byte {
		encryptionKey := make([]byte, 32)
		if _, err := rand.Read(encryptionKey); err != nil {
			log.Panicf("Could not generate encryption key: %v", err)
		}
		return encryptionKey
	}

	JustBeforeEach(func() {
		encrypter = &security.TwoLayerEncrypter{
			Fetcher: fetcher,
		}
	})

	Context("Encrypt", func() {
		Context("When Fetcher returns an error", func() {
			expectedError := fmt.Errorf("Error getting encryption key")
			BeforeEach(func() {
				fetcher = &securityfakes.FakeKeyFetcher{}
				fetcher.GetEncryptionKeyReturns(nil, expectedError)
			})
			It("Should return error", func() {
				encryptedString, err := encrypter.Encrypt(plaintext)
				Expect(encryptedString).To(BeNil())
				Expect(err).To(Equal(expectedError))
			})
		})
		Context("When fetcher return encryption key", func() {
			BeforeEach(func() {
				fetcher = &securityfakes.FakeKeyFetcher{}
				encryptionKey := generateEncryptionKey()
				fetcher.GetEncryptionKeyReturns(encryptionKey, nil)
			})
			It("Should encrypt the data", func() {
				encryptedString, err := encrypter.Encrypt(plaintext)
				Expect(encryptedString).To(Not(BeNil()))
				Expect(err).To(BeNil())
			})
		})
	})

	Context("Decrypt", func() {
		Context("When Fetcher returns an error", func() {
			expectedError := fmt.Errorf("Error getting encryption key")
			BeforeEach(func() {
				fetcher = &securityfakes.FakeKeyFetcher{}
				fetcher.GetEncryptionKeyReturns(nil, expectedError)
			})
			It("Should return error", func() {
				encryptedString, err := encrypter.Decrypt([]byte("cipher"))
				Expect(encryptedString).To(BeNil())
				Expect(err).To(Equal(expectedError))
			})
		})
	})

	Context("When fetcher return encryption key", func() {
		BeforeEach(func() {
			fetcher = &securityfakes.FakeKeyFetcher{}
			encryptionKey := generateEncryptionKey()
			fetcher.GetEncryptionKeyReturns(encryptionKey, nil)
		})
		It("Should decrypt the data", func() {
			encryptedString, _ := encrypter.Encrypt(plaintext)
			decryptedBytes, err := encrypter.Decrypt(encryptedString)
			Expect(decryptedBytes).To(Not(BeNil()))
			Expect(err).To(BeNil())
			Expect(decryptedBytes).To(Equal(plaintext))
		})
	})
})
