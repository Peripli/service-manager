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

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/security"

	"github.com/Peripli/service-manager/pkg/security/securityfakes"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Storage Locker", func() {
	var s *Storage
	var locker *Locker
	var mockdb *sql.DB
	var mock sqlmock.Sqlmock

	var envEncryptionKey []byte

	var fakeEncrypter *securityfakes.FakeEncrypter

	sucessLockRow := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"pg_advisory_lock"}).FromCSVString("true") }
	failLockRow := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"pg_advisory_lock"}).FromCSVString("false") }
	sucessUnlockRow := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"pg_advisory_unlock"}).FromCSVString("true") }

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
		locker = &Locker{
			Storage:       s,
			AdvisoryIndex: 1,
		}
		mock.ExpectQuery(`SELECT CURRENT_DATABASE()`).WillReturnRows(sqlmock.NewRows([]string{"mock"}).FromCSVString("mock"))
		mock.ExpectQuery(`SELECT COUNT(1)*`).WillReturnRows(sqlmock.NewRows([]string{"mock"}).FromCSVString("1"))
		mock.ExpectExec("SELECT pg_advisory_lock*").WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery(`SELECT version, dirty FROM "schema_migrations" LIMIT 1`).WillReturnRows(sqlmock.NewRows([]string{"version", "dirty"}).FromCSVString("20200221152000,false"))
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

	Describe("Lock", func() {
		AfterEach(func() {
			mock.ExpectQuery("SELECT pg_advisory_unlock*").WithArgs(sqlmock.AnyArg()).WillReturnRows(sucessUnlockRow())
			err := locker.Unlock(context.TODO())
			Expect(err).ShouldNot(HaveOccurred())
		})

		BeforeEach(func() {
			mock.ExpectQuery("SELECT pg_advisory_lock*").WillReturnRows(sucessLockRow())
		})

		Context("When lock is already acquired", func() {
			It("Should return an error", func() {
				err := locker.Lock(context.TODO())
				Expect(err).ToNot(HaveOccurred())

				err = locker.Lock(context.TODO())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When lock is not yet acquired", func() {
			It("Should acquire lock", func() {
				err := locker.Lock(context.TODO())
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("TryLock", func() {
		Context("When lock is already acquired by another lock", func() {
			BeforeEach(func() {
				mock.ExpectQuery("SELECT pg_try_advisory_lock*").WillReturnRows(failLockRow())
			})

			It("Should return an error", func() {
				err := locker.Lock(context.TODO())
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Unlock", func() {
		Context("When lock is not acquired", func() {
			It("Should return nil", func() {
				err := locker.Unlock(context.TODO())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("When lock is acquired", func() {
			BeforeEach(func() {
				mock.ExpectQuery("SELECT pg_advisory_lock*").WillReturnRows(sucessLockRow())
				mock.ExpectQuery("SELECT pg_advisory_unlock*").WillReturnRows(sucessUnlockRow())
			})

			It("Should release lock", func() {
				err := locker.Lock(context.TODO())
				Expect(err).ToNot(HaveOccurred())

				err = locker.Unlock(context.TODO())
				Expect(err).To(BeNil())
			})
		})
	})
})
