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
	"errors"
	"io/ioutil"
	"sync"
	"time"

	"github.com/Peripli/service-manager/config"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

var _ = Describe("Postgres Storage State", func() {

	var storageURI string

	BeforeSuite(func() {
		config := config.DefaultSettings()
		appYml, err := ioutil.ReadFile("../../test/common/application.yml")
		Expect(err).ToNot(HaveOccurred())
		err = yaml.Unmarshal(appYml, config)
		Expect(err).ToNot(HaveOccurred())
		storageURI = config.Storage.URI
		Expect(storageURI).ToNot(BeEmpty())
	})

	Describe("Get", func() {
		Context("With valid cache", func() {
			It("should return cached storageError", func() {
				state := newStorageState(errors.New("expected"), time.Now(), nil, time.Second*5)
				err := state.Get()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("expected"))
			})
		})

		Context("With invalid cache", func() {
			It("should return cached storageError", func() {
				db, err := sqlx.Connect(Storage, storageURI)
				if err != nil {
					logrus.Panicln("Could not connect to PostgreSQL:", err)
				}
				state := newStorageState(errors.New("No"), time.Now(), db, time.Second*(-1))
				err = state.Get()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
	Describe("checkDB", func() {
		Context("With valid cache", func() {
			It("should return cached storageError", func() {
				state := newStorageState(errors.New("expected"), time.Now(), nil, time.Second*5)
				err := state.checkDB()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("expected"))
			})
		})
	})
})

func newStorageState(storageError error, lastCheck time.Time, db *sqlx.DB, storageCheckInterval time.Duration) *storageState {
	return &storageState{
		storageError:         storageError,
		lastCheck:            lastCheck,
		mutex:                &sync.RWMutex{},
		db:                   db,
		storageCheckInterval: storageCheckInterval,
	}
}
