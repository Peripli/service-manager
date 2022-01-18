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
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Postgres Storage", func() {
	var scheme *scheme

	BeforeEach(func() {
		scheme = newScheme()
	})

	Describe("Introduce", func() {
		Context("With non-pointer struct", func() {
			It("Should panic", func() {
				intFunc := func() {
					scheme.introduce(postgresEntity{})
				}
				Expect(intFunc).To(Panic())
			})
		})
		Context("With multiple registration of the same entity", func() {
			It("Should panic", func() {
				intFunc := func() {
					scheme.introduce(&postgresEntity{
						storageEntity: &storageEntity{},
					})
				}
				intFunc()
				Expect(intFunc).To(Panic())
			})
		})
	})

	Describe("Provide", func() {
		Context("When no entity for this type is not introduced", func() {
			It("Returns error", func() {
				pgEntity, err := scheme.provide(types.PlatformType)
				Expect(pgEntity).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When the object type does not mach any introduced entity", func() {
			It("Returns error", func() {
				scheme.introduce(&postgresEntity{
					storageEntity: &storageEntity{},
				})
				pgEntity, err := scheme.provide(types.PlatformType)
				Expect(pgEntity).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When introduced entity is not postgres entity", func() {
			It("Should panic", func() {
				intFunc := func() {
					scheme.introduce(&storageEntity{})
				}
				Expect(intFunc).To(Panic())
			})
		})

		Context("When introduced entity is postgres entity", func() {
			It("Returns pg entity", func() {
				scheme.introduce(&postgresEntity{
					storageEntity: &storageEntity{},
				})
				pgEntity, err := scheme.provide(obj{}.GetType())
				Expect(pgEntity).ToNot(BeNil())
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("Convert", func() {
		Context("When no entity for this type is not introduced", func() {
			It("Returns error", func() {
				pgEntity, err := scheme.convert(&obj{})
				Expect(pgEntity).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When the object type does not mach any introduced entity", func() {
			It("Returns error", func() {
				scheme.introduce(&postgresEntity{
					storageEntity: &storageEntity{},
				})
				pgEntity, err := scheme.convert(&dummyTypeObj{})
				Expect(pgEntity).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When FromObject is not ok", func() {
			It("Should panic", func() {
				intFunc := func() {
					scheme.introduce(&pgEntityFromObjectNotOk{
						postgresEntity: &postgresEntity{
							storageEntity: &storageEntity{},
						},
					})
				}
				Expect(intFunc).To(Panic())
			})
		})

		Context("When introduced entity is not postgres entity", func() {
			It("Should panic", func() {
				intFunc := func() {
					scheme.introduce(&storageEntity{})
				}
				Expect(intFunc).To(Panic())
			})
		})

		Context("When introduced entity is postgres entity", func() {
			It("Returns pg entity", func() {
				scheme.introduce(&postgresEntity{
					storageEntity: &storageEntity{},
				})
				pgEntity, err := scheme.convert(&obj{})
				Expect(pgEntity).ToNot(BeNil())
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

type obj struct {
}

func (obj) SetReady(bool) {
}

func (obj) GetReady() bool {
	return true
}

func (obj) Equals(other types.Object) bool {
	return false
}

func (obj) Validate() error {
	return nil
}

func (obj) GetCreatedAt() time.Time {
	return time.Now()
}

func (obj) GetID() string {
	return "id"
}

func (obj) GetLabels() types.Labels {
	return types.Labels{}
}

func (obj) GetType() types.ObjectType {
	return "some type"
}

func (obj) GetUpdatedAt() time.Time {
	return time.Now()
}

func (obj) SetCreatedAt(time time.Time) {
}

func (obj) SetID(id string) {

}

func (obj) SetLabels(labels types.Labels) {
}

func (obj) SetUpdatedAt(time time.Time) {
}

func (obj) GetPagingSequence() int64 {
	return 0
}

func (*obj) SetLastOperation(*types.Operation) {
}

func (*obj) GetLastOperation() *types.Operation {
	return nil
}

type dummyTypeObj struct {
	*obj
}

func (dummyTypeObj) GetType() types.ObjectType {
	return "some other type"
}

type storageEntity struct {
}

func (storageEntity) GetID() string {
	return "id"
}

func (storageEntity) SetID(id string) {

}

func (storageEntity) ToObject() (types.Object, error) {
	return &obj{}, nil
}

func (storageEntity) FromObject(object types.Object) (storage.Entity, error) {
	return &storageEntity{}, nil
}

func (storageEntity) NewLabel(id, entityID, key, value string) storage.Label {
	return nil
}

type postgresEntity struct {
	*storageEntity
}

func (pe postgresEntity) TableName() string {
	return "table"
}

func (postgresEntity) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	return nil, nil
}

func (postgresEntity) LabelEntity() PostgresLabel {
	return nil
}

func (postgresEntity) FromObject(object types.Object) (storage.Entity, error) {
	return &postgresEntity{}, nil
}

type pgEntityFromObjectNotOk struct {
	*postgresEntity
}

func (pgEntityFromObjectNotOk) FromObject(object types.Object) (storage.Entity, error) {
	return nil, nil
}
