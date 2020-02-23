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

package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/security"

	"github.com/Peripli/service-manager/pkg/security/securityfakes"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Secured Storage", func() {
	var s *Storage
	var mockdb *sql.DB
	var mock sqlmock.Sqlmock

	var envEncryptionKey []byte

	var fakeEncrypter *securityfakes.FakeEncrypter

	BeforeEach(func() {
		envEncryptionKey = make([]byte, 32)
		_, err := rand.Read(envEncryptionKey)
		Expect(err).ToNot(HaveOccurred())

		mockdb, mock, err = sqlmock.New()
		Expect(err).ToNot(HaveOccurred())

		s = &Storage{
			ConnectFunc: func(driver string, url string) (*sql.DB, error) {
				return mockdb, nil
			},
		}
		mock.ExpectQuery(`SELECT CURRENT_DATABASE()`).WillReturnRows(sqlmock.NewRows([]string{"mock"}).FromCSVString("mock"))
		mock.ExpectQuery(`SELECT COUNT(1)*`).WillReturnRows(sqlmock.NewRows([]string{"mock"}).FromCSVString("1"))
		mock.ExpectExec("SELECT pg_advisory_lock*").WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery(`SELECT version, dirty FROM "schema_migrations" LIMIT 1`).WillReturnRows(sqlmock.NewRows([]string{"version", "dirty"}).FromCSVString("20200219214200,false"))
		mock.ExpectExec("SELECT pg_advisory_unlock*").WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
		options := storage.DefaultSettings()
		options.EncryptionKey = string(envEncryptionKey)
		options.URI = "sqlmock://sqlmock"
		err = s.Open(options)
		Expect(err).ToNot(HaveOccurred())

		fakeEncrypter = &securityfakes.FakeEncrypter{}

		fakeEncrypter.EncryptCalls(func(ctx context.Context, plainKey []byte, encryptionKey []byte) ([]byte, error) {
			encrypter := &security.AESEncrypter{}
			return encrypter.Encrypt(ctx, plainKey, encryptionKey)
		})

		fakeEncrypter.DecryptCalls(func(ctx context.Context, encryptedKey []byte, encryptionKey []byte) ([]byte, error) {
			encrypter := &security.AESEncrypter{}
			return encrypter.Decrypt(ctx, encryptedKey, encryptionKey)
		})
	})

	AfterEach(func() {
		s.Close()
	})

	Describe("GetEncryptionKey", func() {
		Context("When database returns error when selecting", func() {
			expectedError := fmt.Errorf("expected error")

			BeforeEach(func() {
				mock.ExpectQuery("SELECT").WillReturnError(expectedError)
			})

			It("Should return error", func() {
				encryptionKey, err := s.GetEncryptionKey(context.TODO(), fakeEncrypter.Decrypt)
				Expect(encryptionKey).To(BeNil())
				Expect(err).To(Equal(expectedError))
			})
		})

		Context("When no encryption keys are found", func() {
			BeforeEach(func() {
				rows := sqlmock.NewRows([]string{"secret", "created_at", "updated_at"})
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			})

			It("Should return empty byte array", func() {
				encryptionKey, err := s.GetEncryptionKey(context.TODO(), fakeEncrypter.Decrypt)
				Expect(encryptionKey).To(Not(BeNil()))
				Expect(encryptionKey).To(BeEmpty())
				Expect(err).To(BeNil())
			})
		})

		Context("When encryption key is found", func() {
			plaintext := []byte("secret")

			BeforeEach(func() {
				dbEncryptionKey, _ := fakeEncrypter.Encrypt(context.TODO(), plaintext, envEncryptionKey)
				rows := sqlmock.NewRows([]string{"secret", "created_at", "updated_at"}).
					AddRow(dbEncryptionKey, time.Now(), time.Now())
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			})

			It("Should return decrypted key", func() {
				encryptionKey, err := s.GetEncryptionKey(context.TODO(), fakeEncrypter.Decrypt)
				Expect(encryptionKey).To(Equal(plaintext))
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("SetEncryptionKey", func() {
		Context("When encrypting key returns error", func() {
			It("Should return error", func() {
				err := s.SetEncryptionKey(context.TODO(), []byte{}, fakeEncrypter.Encrypt)
				Expect(err).To(Not(BeNil()))
			})
		})

		Context("When creating query returns error", func() {
			expectedError := fmt.Errorf("expected error")
			BeforeEach(func() {
				mock.ExpectBegin().WillReturnError(nil)
				mock.ExpectExec("INSERT").WillReturnError(expectedError)
			})

			It("Should return error", func() {
				err := s.SetEncryptionKey(context.TODO(), []byte{}, fakeEncrypter.Encrypt)
				Expect(err).To(Not(BeNil()))
			})
		})
		Context("When committing transaction returns error", func() {
			expectedError := fmt.Errorf("expected error")

			BeforeEach(func() {
				mock.ExpectBegin().WillReturnError(nil)
				mock.ExpectExec("INSERT").WillReturnError(nil)
				mock.ExpectCommit().WillReturnError(expectedError)
			})

			It("Should return error", func() {
				err := s.SetEncryptionKey(context.TODO(), []byte{}, fakeEncrypter.Encrypt)
				Expect(err).To(Not(BeNil()))
			})
		})

		Context("When key does not yet exist", func() {
			BeforeEach(func() {
				mock.ExpectPrepare("INSERT").WillReturnError(nil)
				mock.ExpectQuery("INSERT").WillReturnRows(sqlmock.NewRows([]string{"secret"}).FromCSVString("secret"))
			})

			It("Should return nil", func() {
				err := s.SetEncryptionKey(context.TODO(), []byte{}, fakeEncrypter.Encrypt)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("When key already exists", func() {
			BeforeEach(func() {
				mock.ExpectPrepare("INSERT").WillReturnError(nil)
				mock.ExpectQuery("INSERT").WillReturnRows(sqlmock.NewRows([]string{"secret"}).FromCSVString("secret"))
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"secret"}).FromCSVString("secret"))
			})

			It("Should return error", func() {
				err := s.SetEncryptionKey(context.TODO(), []byte{}, fakeEncrypter.Encrypt)
				Expect(err).ToNot(HaveOccurred())
				err = s.SetEncryptionKey(context.TODO(), []byte{}, fakeEncrypter.Encrypt)
				Expect(err).To(HaveOccurred())
			})
		})

	})
})
