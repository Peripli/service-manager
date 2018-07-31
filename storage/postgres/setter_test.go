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
	"crypto/rand"
	"database/sql"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Peripli/service-manager/security"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGresStorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Postgres Storage Suite")
}

var _ = Describe("DbKeySetter", func() {

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
			err := setter.SetEncryptionKey([]byte{})
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
			err := setter.SetEncryptionKey([]byte{})
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
			err := setter.SetEncryptionKey([]byte{})
			Expect(err).To(Not(BeNil()))
		})
	})

	Context("When everything passed", func() {
		BeforeEach(func() {
			result := sqlmock.NewResult(int64(1), int64(1))
			mock.ExpectExec("INSERT").WillReturnResult(result)
		})
		It("Should return nil", func() {
			err := setter.SetEncryptionKey([]byte{})
			Expect(err).To(BeNil())
		})
	})
})
