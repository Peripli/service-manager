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
	"github.com/Peripli/service-manager/pkg/util"
	"strings"
)

type LabelTranslator struct {
	allowedOperations map[string]util.Operation
	allowedSeparators []rune
}

func NewLabelTranslator() util.Translator {
	return &LabelTranslator{
		allowedOperations: map[string]util.Operation{
			"IN": inOp{},
			"=":  eqOp{},
			">":  gtOp{},
			">=": gteOp{},
			"<":  ltOp{},
			"<=": lteOp{},
		},
		allowedSeparators: []rune{
			';',
		},
	}
}

func (l *LabelTranslator) Translate(input string) (string, error) {
	filterStatements := make([]util.FilterStatement, 0)

	rawFilterStatements := strings.FieldsFunc(input, l.split)
	for _, rawStatement := range rawFilterStatements {
		op, err := l.getRawOperation(rawStatement)
		if err != nil {
			return "", err
		}

		filterStatement := l.convertRawStatementToFilterStatement(rawStatement, op)
		if err := filterStatement.Validate(); err != nil {
			return "", err
		}

		filterStatements = append(filterStatements, filterStatement)
	}

	sqlConditions := l.convertFilterStatementsToSQLConditions(filterStatements)
	resultClause := strings.Join(sqlConditions, " AND ")
	return resultClause, nil
}

func (l *LabelTranslator) split(r rune) bool {
	for _, sep := range l.allowedSeparators {
		if r == sep {
			return true
		}
	}
	return false
}

func (l *LabelTranslator) getRawOperation(rawStatement string) (string, error) {
	opIdx := -1
	for _, op := range l.allowedOperations {
		opIdx = strings.Index(rawStatement, op.String())
		if opIdx != -1 {
			return op.String(), nil
		}
	}
	return "", fmt.Errorf("label query operator is missing")
}

func (l *LabelTranslator) convertRawStatementToFilterStatement(rawStatement, op string) util.FilterStatement {
	opIdx := strings.Index(rawStatement, op)
	rightOp := strings.Split(rawStatement[opIdx+len(op):], ",")

	if l.allowedOperations[op].IsMultivalue() {
		rightOp[0] = strings.Trim(rightOp[0], "[")
		rightOp[len(rightOp)-1] = strings.Trim(rightOp[len(rightOp)-1], "]")
	}

	return util.NewFilterStatement(rawStatement[:opIdx], l.allowedOperations[op], rightOp)
}

func (l *LabelTranslator) convertFilterStatementsToSQLConditions(filterStatements []util.FilterStatement) []string {
	conditions := make([]string, 0)
	for _, statement := range filterStatements {
		var value string
		if len(statement.RightOp) > 1 {
			value = fmt.Sprintf("value %s (%s)", statement.Op.String(), strings.Join(statement.RightOp, ","))
		}
		if len(statement.RightOp) == 1 {
			value = fmt.Sprintf("value %s %s", statement.Op.String(), statement.RightOp[0])
		}

		condition := fmt.Sprintf("(key='%s' AND %s)", statement.LeftOp, value)
		conditions = append(conditions, condition)
	}

	return conditions
}
