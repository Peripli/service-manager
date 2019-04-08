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
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type entityProvider func(objectType types.ObjectType) (storage.Entity, bool)
type objectConverter func(object types.Object) (storage.Entity, bool)

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

func (s *scheme) Introduce(entity storage.Entity) {
	obj := entity.ToObject()
	objType := obj.GetType()
	s.instanceProviders = append(s.instanceProviders, func(objectType types.ObjectType) (storage.Entity, bool) {
		if objType != objectType {
			return nil, false
		}
		return entity, true
	})
	s.converters = append(s.converters, func(object types.Object) (storage.Entity, bool) {
		if object.GetType() != objType {
			return nil, false
		}
		return entity.FromObject(object)
	})
}

func (s *scheme) ObjectToEntity(object types.Object) (storage.Entity, bool) {
	for _, c := range s.converters {
		if entity, ok := c(object); ok {
			return entity, true
		}
	}
	return nil, false
}

func (s *scheme) Provide(objectType types.ObjectType) (storage.Entity, bool) {
	for _, v := range s.instanceProviders {
		if entity, ok := v(objectType); ok {
			return entity, true
		}
	}
	return nil, false
}
