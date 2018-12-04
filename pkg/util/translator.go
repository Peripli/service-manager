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

package util

import (
	"errors"
	"fmt"
	"strings"
)

type Translator interface {
	Translate(input string) (string, error)
}

type FilterStatement struct {
	leftOp, op string
	rightOp    []string
}

type LabelTranslator struct {
	allowedOperations map[string]string
	allowedSeparators []rune
}

func NewLabelTranslator() Translator {
	return &LabelTranslator{
		allowedOperations: map[string]string{
			"IN": "IN",
			">":  ">",
			">=": ">=",
			"<=": "<=",
			"<":  "<",
			"=":  "=",
		},
		allowedSeparators: []rune{
			';',
		},
	}
}

func (l *LabelTranslator) Translate(input string) (string, error) {
	filterStatements := make([]FilterStatement, 0)

	rawFilterStatements := strings.FieldsFunc(input, l.split)
	for _, rawStatement := range rawFilterStatements {
		op, err := l.getRawOperation(rawStatement)
		if err != nil {
			return "", err
		}
		statement := l.convertRawToFilterStatement(rawStatement, op)
		filterStatements = append(filterStatements, statement)
	}

	conditions := l.getSQLConditions(filterStatements)

	resultClause := strings.Join(conditions, " AND ")
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
		opIdx = strings.Index(rawStatement, op)
		if opIdx != -1 {
			return op, nil
		}
	}
	return "", errors.New("Invalid label query")
}

func (l *LabelTranslator) convertRawToFilterStatement(rawStatement, op string) FilterStatement {
	opIdx := strings.Index(rawStatement, op)

	rightOp := strings.Split(rawStatement[opIdx+len(op):], ",")
	if len(rightOp) > 1 {
		rightOp[0] = strings.Trim(rightOp[0], "[")
		rightOp[len(rightOp)-1] = strings.Trim(rightOp[len(rightOp)-1], "]")
	}

	return FilterStatement{
		leftOp:  rawStatement[:opIdx],
		op:      op,
		rightOp: rightOp,
	}
}

func (l *LabelTranslator) getSQLConditions(filterStatements []FilterStatement) []string {
	conditions := make([]string, 0)
	for _, statement := range filterStatements {
		values := make([]string, 0)
		for _, value := range statement.rightOp {
			values = append(values, fmt.Sprintf("value %s '%s'", value))
		}
		var value string
		if len(statement.rightOp) > 1 {
			value = fmt.Sprintf("value %s (%s)", l.allowedOperations[statement.op], strings.Join(statement.rightOp, ","))
		}
		if len(statement.rightOp) == 1 {
			value = fmt.Sprintf("value %s %s", l.allowedOperations[statement.op], statement.rightOp[0])
		}

		condition := fmt.Sprintf("(key='%s' AND %s)", statement.leftOp, value)
		conditions = append(conditions, condition)
	}

	return conditions
}
