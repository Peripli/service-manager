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

package query

const (
	// EqualsOperator takes two operands and tests if they are equal
	EqualsOperator eqOperator = "eq"
	// NotEqualsOperator takes two operands and tests if they are not equal
	NotEqualsOperator neOperator = "ne"
	// GreaterThanOperator takes two operands and tests if the left is greater than the right
	GreaterThanOperator gtOperator = "gt"
	// GreaterThanOrEqualOperator takes two operands and tests if the left is greater than or equal the right
	GreaterThanOrEqualOperator geOperator = "ge"
	// Contains takes two operands and tests if the left contains the right
	ContainsOperator containsOperator = "contains"
	// LessThanOperator takes two operands and tests if the left is lesser than the right
	LessThanOperator ltOperator = "lt"
	// LessThanOrEqualOperator takes two operands and tests if the left is lesser than or equal the right
	LessThanOrEqualOperator leOperator = "le"
	// InOperator takes two operands and tests if the left is contained in the right
	InOperator         inOperator         = "in"
	InSubqueryOperator inSubqueryOperator = "inSubquery"
	// NotInOperator takes two operands and tests if the left is not contained in the right
	NotInOperator notInOperator = "notin"
	// NotExistsSubquery receives a sub-query as single left-operand and checks the sub-query for rows existence. If there're no rows then it will return TRUE, otherwise FALSE.
	// Applicable for usage only with ExistQuery Criterion type
	NotExistsSubquery notExistsSubquery = "notexists"
	// ExistsSubquery receives a sub-query as single left-operand and checks the sub-query for rows existence. If there are any, then it will return TRUE otherwise FALSE.
	// Applicable for usage only with ExistQuery Criterion type
	ExistsSubquery existsSubquery = "exists"
	// EqualsOrNilOperator takes two operands and tests if the left is equal to the right, or if the left is nil
	EqualsOrNilOperator enOperator = "en"

	NoOperator noOperator = "nop"
)

type eqOperator string

func (o eqOperator) String() string {
	return string(o)
}

func (eqOperator) IsNumeric() bool {
	return false
}

func (eqOperator) Type() OperatorType {
	return UnivariateOperator
}

func (eqOperator) IsNullable() bool {
	return false
}

type neOperator string

func (o neOperator) String() string {
	return string(o)
}

func (neOperator) Type() OperatorType {
	return UnivariateOperator
}

func (neOperator) IsNullable() bool {
	return false
}

func (neOperator) IsNumeric() bool {
	return false
}

type gtOperator string

func (o gtOperator) String() string {
	return string(o)
}

func (gtOperator) Type() OperatorType {
	return UnivariateOperator
}

func (gtOperator) IsNullable() bool {
	return false
}

func (gtOperator) IsNumeric() bool {
	return true
}

type ltOperator string

func (o ltOperator) String() string {
	return string(o)
}

func (ltOperator) Type() OperatorType {
	return UnivariateOperator
}

func (ltOperator) IsNullable() bool {
	return false
}

func (ltOperator) IsNumeric() bool {
	return true
}

type inOperator string

func (o inOperator) String() string {
	return string(o)
}

func (inOperator) Type() OperatorType {
	return MultivariateOperator
}

func (inOperator) IsNullable() bool {
	return false
}

func (inOperator) IsNumeric() bool {
	return false
}

type notInOperator string

func (o notInOperator) String() string {
	return string(o)
}

func (notInOperator) Type() OperatorType {
	return MultivariateOperator
}

func (notInOperator) IsNullable() bool {
	return false
}

func (notInOperator) IsNumeric() bool {
	return false
}

type containsOperator string

func (o containsOperator) String() string {
	return string(o)
}

func (containsOperator) Type() OperatorType {
	return UnivariateOperator
}

func (containsOperator) IsNullable() bool {
	return false
}

func (containsOperator) IsNumeric() bool {
	return false
}

type inSubqueryOperator string

func (o inSubqueryOperator) String() string {
	return string(o)
}

func (inSubqueryOperator) Type() OperatorType {
	return UnivariateOperator
}

func (inSubqueryOperator) IsNullable() bool {
	return false
}

func (inSubqueryOperator) IsNumeric() bool {
	return false
}

type existsSubquery string

func (o existsSubquery) String() string {
	return string(o)
}

func (existsSubquery) Type() OperatorType {
	return UnivariateOperator
}

func (existsSubquery) IsNullable() bool {
	return false
}

func (existsSubquery) IsNumeric() bool {
	return false
}

type notExistsSubquery string

func (o notExistsSubquery) String() string {
	return string(o)
}

func (notExistsSubquery) Type() OperatorType {
	return UnivariateOperator
}

func (notExistsSubquery) IsNullable() bool {
	return false
}

func (notExistsSubquery) IsNumeric() bool {
	return false
}

type enOperator string

func (o enOperator) String() string {
	return string(o)
}

func (enOperator) Type() OperatorType {
	return UnivariateOperator
}

func (enOperator) IsNullable() bool {
	return true
}

func (enOperator) IsNumeric() bool {
	return false
}

type geOperator string

func (o geOperator) String() string {
	return string(o)
}

func (geOperator) Type() OperatorType {
	return UnivariateOperator
}

func (geOperator) IsNullable() bool {
	return false
}

func (geOperator) IsNumeric() bool {
	return true
}

type leOperator string

func (o leOperator) String() string {
	return string(o)
}

func (leOperator) Type() OperatorType {
	return UnivariateOperator
}

func (leOperator) IsNullable() bool {
	return false
}

func (leOperator) IsNumeric() bool {
	return true
}

type noOperator string

func (o noOperator) String() string {
	return string(o)
}

func (noOperator) Type() OperatorType {
	return MultivariateOperator
}

func (noOperator) IsNullable() bool {
	return false
}

func (noOperator) IsNumeric() bool {
	return false
}
