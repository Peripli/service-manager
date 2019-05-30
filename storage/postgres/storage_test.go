/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package postgres

import (
	"context"
	"database/sql"

	"github.com/Peripli/service-manager/storage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Postgres Storage", func() {
	pgStorage := &Storage{
		ConnectFunc: func(driver string, url string) (*sql.DB, error) {
			return sql.Open(driver, url)
		},
	}

	Describe("Lock", func() {
		Context("Called with uninitialized db", func() {
			It("Should panic", func() {
				Expect(func() { pgStorage.Lock(context.TODO()) }).To(Panic())
			})
		})
	})

	Context("Unlock", func() {
		Context("Called with uninitialized db", func() {
			It("Should panic", func() {
				Expect(func() { pgStorage.Unlock(context.TODO()) }).To(Panic())
			})
		})
	})

	Context("GetEncryptionKey", func() {
		Context("Called with uninitialized db", func() {
			It("Should panic", func() {
				Expect(func() {
					pgStorage.GetEncryptionKey(context.TODO(), func(i context.Context, bytes3 []byte, bytes2 []byte) (bytes []byte, e error) {
						return []byte{}, nil
					})
				}).To(Panic())
			})
		})
	})

	Context("SetEncryptionKey", func() {
		Context("Called with uninitialized db", func() {
			It("Should panic", func() {
				Expect(func() {
					pgStorage.SetEncryptionKey(context.TODO(), []byte{}, func(i context.Context, bytes3 []byte, bytes2 []byte) (bytes []byte, e error) {
						return []byte{}, nil
					})
				}).To(Panic())
			})
		})
	})

	Describe("Ping", func() {
		Context("Called with uninitialized db", func() {
			It("Should panic", func() {
				Expect(func() { pgStorage.Ping() }).To(Panic())
			})
		})
	})

	Describe("SelectContext", func() {
		Context("Called with uninitialized db", func() {
			It("Should panic", func() {
				Expect(func() { pgStorage.SelectContext(context.Background(), nil, "") }).To(Panic())
			})
		})
	})

	Describe("Open", func() {
		AfterEach(func() {
			pgStorage.db.Close()
			pgStorage.db = nil
		})

		Context("Called with empty uri", func() {
			It("Should return error", func() {
				err := pgStorage.Open(&storage.Settings{
					URI:           "",
					MigrationsURL: "file://migrations",
				})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Called with invalid postgres uri", func() {
			It("Should return error", func() {
				err := pgStorage.Open(&storage.Settings{
					URI:               "invalid",
					MigrationsURL:     "invalid",
					EncryptionKey:     "ejHjRNHbS0NaqARSRvnweVV9zcmhQEa8",
					SkipSSLValidation: true,
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Close", func() {
		Context("Called with uninitialized db", func() {
			It("Should not panic", func() {
				Expect(func() { pgStorage.Close() }).ToNot(Panic())
			})
		})
	})

})
