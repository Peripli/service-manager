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
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/pkg/web"
)

type LabelTranslator struct{}

func (l *LabelTranslator) Translate(segments []web.QuerySegment) (string, error) {
	sqlConditions := l.convertFilterStatementsToSQLConditions(segments)
	resultClause := strings.Join(sqlConditions, " AND ")
	return resultClause, nil
}

func (l *LabelTranslator) convertFilterStatementsToSQLConditions(filterStatements []web.QuerySegment) []string {
	conditions := make([]string, 0)
	for _, statement := range filterStatements {
		var value string
		if statement.Operator.IsMultiVariate() {
			value = fmt.Sprintf("value %s (%s)", statement.Operator, strings.Join(statement.RightOp, ","))
		} else {
			value = fmt.Sprintf("value %s %s", statement.Operator, statement.RightOp[0])
		}

		condition := fmt.Sprintf("(key='%s' AND %s)", statement.LeftOp, value)
		conditions = append(conditions, condition)
	}

	return conditions
}
