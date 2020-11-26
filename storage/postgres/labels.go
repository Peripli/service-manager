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
	sqlxtypes "github.com/jmoiron/sqlx/types"
	"reflect"
	"strings"
	"time"

	"github.com/Peripli/service-manager/pkg/query"
)

func findTagType(tags []tagType, tagName string) reflect.Type {
	for _, tag := range tags {
		if strings.Split(tag.Tag, ",")[0] == tagName {
			return tag.Type
		}
	}
	return nil
}

var (
	intType       = reflect.TypeOf(int(1))
	int64Type     = reflect.TypeOf(int64(1))
	timeType      = reflect.TypeOf(time.Time{})
	byteSliceType = reflect.TypeOf([]byte{})
	jsonType      = reflect.TypeOf(sqlxtypes.JSONText{})
)

func determineCastByType(tagType reflect.Type) string {
	dbCast := ""
	switch tagType {
	case intType:
		fallthrough
	case int64Type:
		fallthrough
	case timeType:
		fallthrough
	case byteSliceType:
		dbCast = ""
	default:
		dbCast = "::text"
	}
	return dbCast
}

func hasMultiVariateOp(criteria []query.Criterion) bool {
	for _, opt := range criteria {
		if opt.Operator.Type() == query.MultivariateOperator {
			return true
		}
	}
	return false
}
