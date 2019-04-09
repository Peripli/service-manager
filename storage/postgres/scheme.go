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

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type entityProvider func(objectType types.ObjectType) (PostgresEntity, bool, error)
type objectConverter func(object types.Object) (PostgresEntity, bool, error)

func newScheme() *scheme {
	return &scheme{
		instanceProviders: make([]entityProvider, 0),
		converters:        make([]objectConverter, 0),
	}
}

type scheme struct {
	instanceProviders []entityProvider
	converters        []objectConverter
}

func (s *scheme) introduce(entity storage.Entity) {
	obj := entity.ToObject()
	objType := obj.GetType()
	s.instanceProviders = append(s.instanceProviders, func(objectType types.ObjectType) (PostgresEntity, bool, error) {
		if objType != objectType {
			return nil, false, nil
		}
		pgEntity, ok := entity.(PostgresEntity)
		if !ok {
			return nil, false, fmt.Errorf("no postgres entity is introduced for object of type %s", objectType)
		}
		return pgEntity, true, nil
	})
	s.converters = append(s.converters, func(object types.Object) (PostgresEntity, bool, error) {
		objectType := object.GetType()
		if objectType != objType {
			return nil, false, nil
		}
		entityFromObject, ok := entity.FromObject(object)
		if !ok {
			return nil, false, nil
		}
		pgEntity, ok := entityFromObject.(PostgresEntity)
		if !ok {
			return nil, false, fmt.Errorf("no postgres entity is introduced for object of type %s", objectType)
		}
		return pgEntity, true, nil
	})
}

func (s *scheme) convert(object types.Object) (PostgresEntity, error) {
	for _, c := range s.converters {
		entity, ok, err := c(object)
		if err != nil {
			return nil, err
		}
		if ok {
			return entity, nil
		}
	}
	return nil, fmt.Errorf("no postgres entity is introduced for object of type %s", object.GetType())
}

func (s *scheme) provide(objectType types.ObjectType) (PostgresEntity, error) {
	for _, v := range s.instanceProviders {
		entity, ok, err := v(objectType)
		if err != nil {
			return nil, err
		}
		if ok {
			return entity, nil
		}
	}
	return nil, fmt.Errorf("no postgres entity is introduced for object of type %s", objectType)
}
