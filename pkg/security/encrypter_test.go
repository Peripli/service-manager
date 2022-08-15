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
	"context"
	"crypto/rand"
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security"
)

var _ = Describe("AES Encrypter", func() {
	var encrypter security.Encrypter
	plaintext := []byte("plaintext")

	generateEncryptionKey := func() []byte {
		encryptionKey := make([]byte, 32)
		if _, err := rand.Read(encryptionKey); err != nil {
			log.Panicf("Could not generate encryption key: %v", err)
		}
		return encryptionKey
	}

	BeforeEach(func() {
		encrypter = &security.AESEncrypter{}
	})

	Context("Encrypt", func() {
		Context("when the key is malformed", func() {
			It("returns the error", func() {
				encryptionKey := []byte{}
				_, err := encrypter.Encrypt(context.TODO(), plaintext, encryptionKey)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("when the key is valid", func() {
			It("encrypts the data", func() {
				encryptionKey := generateEncryptionKey()
				encryptedString, err := encrypter.Encrypt(context.TODO(), plaintext, encryptionKey)
				Expect(encryptedString).To(Not(BeNil()))
				Expect(err).To(BeNil())
			})
		})
	})

	Context("Decrypt", func() {
		Context("when the key is malformed", func() {
			It("returns the error", func() {
				encryptionKey := []byte{}
				_, err := encrypter.Decrypt(context.TODO(), plaintext, encryptionKey)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("when the key is valid", func() {
			It("decrypts the data", func() {
				encryptionKey := generateEncryptionKey()
				encryptedString, _ := encrypter.Encrypt(context.TODO(), plaintext, encryptionKey)
				decryptedBytes, err := encrypter.Decrypt(context.TODO(), encryptedString, encryptionKey)
				Expect(decryptedBytes).To(Not(BeNil()))
				Expect(err).To(BeNil())
				Expect(decryptedBytes).To(Equal(plaintext))
			})
		})
	})
})
