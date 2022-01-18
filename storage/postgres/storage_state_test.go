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
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Postgres Storage State", func() {

	var (
		mockDB *sql.DB
		sqlxDB *sqlx.DB
		mock   sqlmock.Sqlmock

		state *storageState
	)
	BeforeEach(func() {
		var err error
		mockDB, mock, err = sqlmock.New()
		Expect(err).To(Not(HaveOccurred()))
		sqlxDB = sqlx.NewDb(mockDB, "sqlmock")

		state = &storageState{
			lastCheckTime:        time.Now(),
			mutex:                &sync.RWMutex{},
			db:                   sqlxDB,
			storageCheckInterval: time.Second * 1,
		}
	})

	AfterEach(func() {
		mockDB.Close()
		sqlxDB.Close()
	})

	Describe("Get", func() {
		Context("when cache has not yet expired", func() {
			It("returns nil", func() {
				Expect(state.Get()).To(Not(HaveOccurred()))
			})
		})

		Context("when cache has expired", func() {
			Context("when storage query fails", func() {
				BeforeEach(func() {
					mock.ExpectQuery("^SELECT 1$").WillReturnError(fmt.Errorf("error"))
				})

				It("returns error", func() {
					Eventually(func() error { return state.Get() }, time.Second*2).Should(Equal(fmt.Errorf("error")))
				})
			})

			Context("when storage query succeeds", func() {
				BeforeEach(func() {
					mock.ExpectQuery("^SELECT 1$").WillReturnRows(sqlmock.NewRows([]string{"1"}))

				})

				It("returns nil", func() {
					Eventually(func() error { return state.Get() }, time.Second*2).ShouldNot(HaveOccurred())
				})
			})
		})
	})
})
