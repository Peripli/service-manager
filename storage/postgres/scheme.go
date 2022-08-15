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
	"fmt"
	"reflect"

	"github.com/gofrs/uuid"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

type entityProvider func() (PostgresEntity, error)
type objectConverter func(object types.Object) (PostgresEntity, error)

func newScheme() *scheme {
	return &scheme{
		instanceProviders:           make(map[types.ObjectType]entityProvider),
		converters:                  make(map[types.ObjectType]objectConverter),
		entityToObjectTypeConverter: make(map[string]string),
	}
}

type scheme struct {
	instanceProviders           map[types.ObjectType]entityProvider
	converters                  map[types.ObjectType]objectConverter
	entityToObjectTypeConverter map[string]string
}

func (s *scheme) introduce(entity storage.Entity) {
	t := reflect.TypeOf(entity)
	if t.Kind() != reflect.Ptr {
		panic("All entities must be pointers to structs.")
	}
	obj, err := entity.ToObject()
	if err != nil {
		panic(fmt.Errorf("could introduce entity: %s", err))
	}
	objType := obj.GetType()

	_, providerAlreadyExists := s.instanceProviders[objType]
	_, converterAlreadyExists := s.converters[objType]
	if providerAlreadyExists || converterAlreadyExists {
		panic(fmt.Sprintf("Entity for object with type %s has already been introduced", objType))
	}
	s.converters[objType] = func(object types.Object) (PostgresEntity, error) {
		entityFromObject, err := entity.FromObject(object)
		if err != nil {
			return nil, fmt.Errorf("regsitered entity cannot convert object from type %s: %s", object.GetType(), err)
		}
		pgEntity, ok := entityFromObject.(PostgresEntity)
		if !ok {
			return nil, fmt.Errorf("no postgres entity is introduced for object of type %s", object.GetType())
		}
		return pgEntity, nil
	}
	s.instanceProviders[objType] = func() (PostgresEntity, error) {
		object, err := entity.ToObject()
		if err != nil {
			return nil, fmt.Errorf("could not provide postgres entity for type %s: %s", objType, err)
		}
		return s.convert(object)
	}

	pgEntity, err := s.instanceProviders[objType]()
	if err != nil {
		panic(fmt.Sprintf("Unable to construct PostgresEntity when introducing object type %s: %s", objType.String(), err))
	}
	s.entityToObjectTypeConverter[pgEntity.TableName()] = objType.String()
}

func (s *scheme) convert(object types.Object) (PostgresEntity, error) {
	objectType := object.GetType()
	converter, exists := s.converters[objectType]
	if !exists {
		return nil, fmt.Errorf("no postgres entity is introduced for object of type %s", objectType)
	}
	return converter(object)
}

func (s *scheme) provide(objectType types.ObjectType) (PostgresEntity, error) {
	provider, exists := s.instanceProviders[objectType]
	if !exists {
		return nil, fmt.Errorf("no postgres entity is introduced for object of type %s", objectType)
	}
	return provider()
}

func (s *scheme) provideLabel(objectType types.ObjectType, objectID, key, value string) (PostgresLabel, error) {
	provider, exists := s.instanceProviders[objectType]
	if !exists {
		return nil, fmt.Errorf("no postgres entity is introduced for object of type %s", objectType)
	}
	entity, err := provider()
	if err != nil {
		return nil, err
	}
	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for label: %s", err)
	}
	label := entity.NewLabel(UUID.String(), objectID, key, value)
	pgLabel, ok := label.(PostgresLabel)
	if !ok {
		return nil, fmt.Errorf("postgres storage requires labels to implement LabelEntity, got %T", label)
	}

	return pgLabel, nil
}
