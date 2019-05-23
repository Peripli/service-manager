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

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type entityProvider func() (PostgresEntity, error)
type objectConverter func(object types.Object) (PostgresEntity, error)

func newScheme() *scheme {
	return &scheme{
		instanceProviders: make(map[types.ObjectType]entityProvider),
		converters:        make(map[types.ObjectType]objectConverter),
	}
}

type scheme struct {
	instanceProviders map[types.ObjectType]entityProvider
	converters        map[types.ObjectType]objectConverter
}

func (s *scheme) introduce(entity storage.Entity) {
	t := reflect.TypeOf(entity)
	if t.Kind() != reflect.Ptr {
		panic("All entities must be pointers to structs.")
	}
	obj := entity.ToObject()
	objType := obj.GetType()

	_, providerAlreadyExists := s.instanceProviders[objType]
	_, converterAlreadyExists := s.converters[objType]
	if providerAlreadyExists || converterAlreadyExists {
		panic(fmt.Sprintf("Entity for object with type %s has already been introduced", objType))
	}
	s.converters[objType] = func(object types.Object) (PostgresEntity, error) {
		entityFromObject, ok := entity.FromObject(object)
		if !ok {
			return nil, fmt.Errorf("regsitered entity cannot convert object from type %s", object.GetType())
		}
		pgEntity, ok := entityFromObject.(PostgresEntity)
		if !ok {
			return nil, fmt.Errorf("no postgres entity is introduced for object of type %s", object.GetType())
		}
		return pgEntity, nil
	}
	s.instanceProviders[objType] = func() (PostgresEntity, error) {
		return s.convert(entity.ToObject())
	}
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
