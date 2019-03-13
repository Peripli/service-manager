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

package storage

import (
	"github.com/Peripli/service-manager/pkg/types"
)

type EntityProvider func(objectType types.ObjectType) (Entity, bool)
type ObjectConverter func(object types.Object) (Entity, bool)

func NewScheme() *Scheme {
	return &Scheme{
		instanceProviders: make([]EntityProvider, 0),
		converters:        make([]ObjectConverter, 0),
	}
}

type Scheme struct {
	instanceProviders []EntityProvider
	converters        []ObjectConverter
}

func (s *Scheme) Introduce(entity Entity) {
	obj := entity.ToObject()
	objType := obj.GetType()
	s.instanceProviders = append(s.instanceProviders, func(objectType types.ObjectType) (Entity, bool) {
		if objType != objectType {
			return nil, false
		}
		return entity, true
	})
	s.converters = append(s.converters, func(object types.Object) (Entity, bool) {
		if object.GetType() != objType {
			return nil, false
		}
		return entity.FromObject(object)
	})
}

func (s *Scheme) ObjectToEntity(object types.Object) (Entity, bool) {
	for _, c := range s.converters {
		if entity, ok := c(object); ok {
			return entity, true
		}
	}
	return nil, false
}

func (s *Scheme) Provide(objectType types.ObjectType) (Entity, bool) {
	for _, v := range s.instanceProviders {
		if entity, ok := v(objectType); ok {
			return entity, true
		}
	}
	return nil, false
}

func (s *Scheme) StorageLabelsToType(labels []Label) types.Labels {
	labelValues := make(map[string][]string)
	for _, label := range labels {
		values, exists := labelValues[label.GetKey()]
		if exists {
			labelValues[label.GetKey()] = append(values, label.GetValue())
		} else {
			labelValues[label.GetKey()] = []string{label.GetValue()}
		}
	}
	return labelValues
}
