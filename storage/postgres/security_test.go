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
	"sync"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Security", func() {

	Describe("KeyFetcher", func() {
		var fetcher security.KeyFetcher
		var mockdb *sql.DB
		var mock sqlmock.Sqlmock

		envEncryptionKey := make([]byte, 32)

		JustBeforeEach(func() {
			fetcher = &keyFetcher{
				db:            sqlx.NewDb(mockdb, "sqlmock"),
				encryptionKey: envEncryptionKey,
			}
		})
		BeforeEach(func() {
			mockdb, mock, _ = sqlmock.New()
			rand.Read(envEncryptionKey)
		})
		AfterEach(func() {
			mockdb.Close()
		})

		Context("When database returns error when selecting", func() {
			expectedError := fmt.Errorf("expected error")
			BeforeEach(func() {
				mock.ExpectQuery("SELECT").WillReturnError(expectedError)
			})
			It("Should return error", func() {
				encryptionKey, err := fetcher.GetEncryptionKey(context.TODO())
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
				encryptionKey, err := fetcher.GetEncryptionKey(context.TODO())
				Expect(encryptionKey).To(Not(BeNil()))
				Expect(encryptionKey).To(BeEmpty())
				Expect(err).To(BeNil())
			})
		})

		Context("When encryption key is found", func() {
			plaintext := []byte("secret")

			BeforeEach(func() {
				dbEncryptionKey, _ := security.Encrypt(plaintext, envEncryptionKey)
				rows := sqlmock.NewRows([]string{"secret", "created_at", "updated_at"}).
					AddRow(dbEncryptionKey, time.Now(), time.Now())
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			})
			It("Should return decrypted key", func() {
				encryptionKey, err := fetcher.GetEncryptionKey(context.TODO())
				Expect(encryptionKey).To(Equal(plaintext))
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("KeySetter", func() {

		var setter security.KeySetter
		var mockdb *sql.DB
		var mock sqlmock.Sqlmock

		envEncryptionKey := make([]byte, 32)

		JustBeforeEach(func() {
			setter = &keySetter{
				db:            sqlx.NewDb(mockdb, "sqlmock"),
				encryptionKey: envEncryptionKey,
			}
		})
		BeforeEach(func() {
			mockdb, mock, _ = sqlmock.New()
			rand.Read(envEncryptionKey)
		})
		AfterEach(func() {
			mockdb.Close()
		})

		Context("When encrypting key returns error", func() {
			It("Should return error", func() {
				err := setter.SetEncryptionKey(context.TODO(), []byte{})
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
				err := setter.SetEncryptionKey(context.TODO(), []byte{})
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
				err := setter.SetEncryptionKey(context.TODO(), []byte{})
				Expect(err).To(Not(BeNil()))
			})
		})

		Context("When everything passed", func() {
			BeforeEach(func() {
				result := sqlmock.NewResult(int64(1), int64(1))
				mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"secret", "created_at", "updated_at"}))
				mock.ExpectExec("INSERT").WillReturnResult(result)
			})
			It("Should return nil", func() {
				err := setter.SetEncryptionKey(context.TODO(), []byte{})
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("Locking", func() {
		var mockdb *sql.DB
		var mock sqlmock.Sqlmock
		var storage *securityStorage
		envEncryptionKey := make([]byte, 32)

		JustBeforeEach(func() {
			storage = &securityStorage{
				db:            sqlx.NewDb(mockdb, "sqlmock"),
				encryptionKey: envEncryptionKey,
				isLocked:      false,
				mutex:         &sync.Mutex{},
			}
		})
		BeforeEach(func() {
			mockdb, mock, _ = sqlmock.New()
			rand.Read(envEncryptionKey)
		})
		AfterEach(func() {
			mockdb.Close()
		})

		Describe("Lock", func() {

			Context("When lock is already acquired", func() {
				It("Should return an error", func() {
					storage.isLocked = true
					err := storage.Lock(context.TODO())
					Expect(err).ToNot(BeNil())
				})
			})

			Context("When lock is not yet acquired", func() {
				AfterEach(func() {
					storage.Unlock(context.TODO())
				})
				BeforeEach(func() {
					mock.ExpectExec("SELECT").WillReturnResult(sqlmock.NewResult(int64(1), int64(1)))
				})
				It("Should acquire lock", func() {
					err := storage.Lock(context.TODO())
					Expect(err).To(BeNil())
					Expect(storage.isLocked).To(Equal(true))
				})
			})
		})

		Describe("Unlock", func() {
			Context("When lock is not acquired", func() {
				It("Should return nil", func() {
					storage.isLocked = false
					err := storage.Unlock(context.TODO())
					Expect(err).To(BeNil())
				})
			})
			Context("When lock is acquired", func() {
				BeforeEach(func() {
					mock.ExpectExec("SELECT").WillReturnResult(sqlmock.NewResult(int64(1), int64(1)))
				})
				It("Should release lock", func() {
					storage.isLocked = true
					err := storage.Unlock(context.TODO())
					Expect(err).To(BeNil())
					Expect(storage.isLocked).To(Equal(false))
				})
			})
		})
	})
})
