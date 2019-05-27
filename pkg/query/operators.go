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
	NotEqualsOperator neqOperator = "neq"
	// GreaterThanOperator takes two operands and tests if the left is greater than the right
	GreaterThanOperator gtOperator = "gt"
	// GreaterThanOrEqualOperator takes two operands and tests if the left is greater than or equal the right
	GreaterThanOrEqualOperator gteOperator = "gte"
	// LessThanOperator takes two operands and tests if the left is lesser than the right
	LessThanOperator ltOperator = "lt"
	// LessThanOrEqualOperator takes two operands and tests if the left is lesser than or equal the right
	LessThanOrEqualOperator lteOperator = "lte"
	// InOperator takes two operands and tests if the left is contained in the right
	InOperator inOperator = "in"
	// NotInOperator takes two operands and tests if the left is not contained in the right
	NotInOperator notInOperator = "notin"
	// EqualsOrNilOperator takes two operands and tests if the left is equal to the right, or if the left is nil
	EqualsOrNilOperator eqOrNilOperator = "eqornil"
)

type eqOperator string

func (o eqOperator) String() string {
	return string(o)
}

func (eqOperator) RequiresNumber() bool {
	return false
}

func (eqOperator) Type() OperatorType {
	return UnivariateOperator
}

func (eqOperator) IsNullable() bool {
	return false
}

type neqOperator string

func (o neqOperator) String() string {
	return string(o)
}

func (neqOperator) Type() OperatorType {
	return UnivariateOperator
}

func (neqOperator) IsNullable() bool {
	return false
}

func (neqOperator) RequiresNumber() bool {
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

func (gtOperator) RequiresNumber() bool {
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

func (ltOperator) RequiresNumber() bool {
	return true
}

type inOperator string

func (o inOperator) String() string {
	return string(o)
}

func (inOperator) Type() OperatorType {
	return MultivareateOperator
}

func (inOperator) IsNullable() bool {
	return false
}

func (inOperator) RequiresNumber() bool {
	return false
}

type notInOperator string

func (o notInOperator) String() string {
	return string(o)
}

func (notInOperator) Type() OperatorType {
	return MultivareateOperator
}

func (notInOperator) IsNullable() bool {
	return false
}

func (notInOperator) RequiresNumber() bool {
	return false
}

type eqOrNilOperator string

func (o eqOrNilOperator) String() string {
	return string(o)
}

func (eqOrNilOperator) Type() OperatorType {
	return UnivariateOperator
}

func (eqOrNilOperator) IsNullable() bool {
	return true
}

func (eqOrNilOperator) RequiresNumber() bool {
	return false
}

type gteOperator string

func (o gteOperator) String() string {
	return string(o)
}

func (gteOperator) Type() OperatorType {
	return UnivariateOperator
}

func (gteOperator) IsNullable() bool {
	return false
}

func (gteOperator) RequiresNumber() bool {
	return true
}

type lteOperator string

func (o lteOperator) String() string {
	return string(o)
}

func (lteOperator) Type() OperatorType {
	return UnivariateOperator
}

func (lteOperator) IsNullable() bool {
	return false
}

func (lteOperator) RequiresNumber() bool {
	return true
}
