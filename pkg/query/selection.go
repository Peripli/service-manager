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

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Peripli/service-manager/pkg/util/slice"

	"github.com/Peripli/service-manager/pkg/web"
)

// Operator is a query operator
type Operator string

const (
	// EqualsOperator takes two operands and tests if they are equal
	EqualsOperator Operator = "="
	// NotEqualsOperator takes two operands and tests if they are not equal
	NotEqualsOperator Operator = "!="
	// GreaterThanOperator takes two operands and tests if the left is greater than the right
	GreaterThanOperator Operator = "gt"
	// LessThanOperator takes two operands and tests if the left is lesser than the right
	LessThanOperator Operator = "lt"
	// InOperator takes two operands and tests if the left is contained in the right
	InOperator Operator = "in"
	// NotInOperator takes two operands and tests if the left is not contained in the right
	NotInOperator Operator = "notin"
	// EqualsOrNilOperator takes two operands and tests if the left is equal to the right, or if the left is nil
	EqualsOrNilOperator Operator = "eqornil"
)

// IsMultiVariate returns true if the operator requires right operand with multiple values
func (op Operator) IsMultiVariate() bool {
	return op == InOperator || op == NotInOperator
}

// IsNullable returns true if the operator can check if the left operand is nil
func (op Operator) IsNullable() bool {
	return op == EqualsOrNilOperator
}

// IsNumeric returns true if the operator works only with numeric operands
func (op Operator) IsNumeric() bool {
	return op == LessThanOperator || op == GreaterThanOperator
}

var operators = []Operator{EqualsOperator, NotEqualsOperator, InOperator,
	NotInOperator, GreaterThanOperator, LessThanOperator, EqualsOrNilOperator}

const (
	// OpenBracket is the token that denotes the beginning of a multivariate operand
	OpenBracket string = "["
	// OpenBracket is the token that denotes the end of a multivariate operand
	CloseBracket string = "]"
	// Separator is the separator between field and label queries
	Separator        rune   = ','
	OperandSeparator string = "+"
)

// CriterionType is a type of criteria to be applied when querying
type CriterionType string

const (
	// FieldQuery denotes that the query should be executed on the entity's fields
	FieldQuery CriterionType = "fieldQuery"
	// LabelQuery denotes that the query should be executed on the entity's labels
	LabelQuery CriterionType = "labelQuery"
)

var supportedQueryTypes = []CriterionType{FieldQuery, LabelQuery}

// UnsupportedQueryError is an error to show that the provided query cannot be executed
type UnsupportedQueryError struct {
	Message string
}

func (uq *UnsupportedQueryError) Error() string {
	return uq.Message
}

// Criterion is a single part of a query criteria
type Criterion struct {
	// LeftOp is the left operand in the query
	LeftOp string
	// Operator is the query operator
	Operator Operator
	// RightOp is the right operand in the query which can be multivariate
	RightOp []string
	// Type is the type of the query
	Type CriterionType
}

// ByField constructs a new criterion for field querying
func ByField(operator Operator, leftOp string, rightOp ...string) Criterion {
	return newCriterion(leftOp, operator, rightOp, FieldQuery)
}

// ByLabel constructs a new criterion for label querying
func ByLabel(operator Operator, leftOp string, rightOp ...string) Criterion {
	return newCriterion(leftOp, operator, rightOp, LabelQuery)
}

func newCriterion(leftOp string, operator Operator, rightOp []string, criteriaType CriterionType) Criterion {
	return Criterion{LeftOp: leftOp, Operator: operator, RightOp: rightOp, Type: criteriaType}
}

func (c Criterion) Validate() error {
	if len(c.RightOp) > 1 && !c.Operator.IsMultiVariate() {
		return fmt.Errorf("multiple values received for single value operation")
	}
	if c.Operator.IsNullable() && c.Type != FieldQuery {
		return &UnsupportedQueryError{"nullable operations are supported only for field queries"}
	}
	if c.Operator.IsNumeric() && !isNumeric(c.RightOp[0]) {
		return &UnsupportedQueryError{Message: fmt.Sprintf("%s is numeric operator, but the right operand is not numeric", c.Operator)}
	}
	if slice.StringsAnyEquals(c.RightOp, "") {
		return &UnsupportedQueryError{Message: "right operand must have value"}
	}
	return nil
}

func mergeCriteria(c1 []Criterion, c2 []Criterion) ([]Criterion, error) {
	result := c1
	fieldQueryLeftOperands := make(map[string]bool)
	for _, criterion := range c1 {
		if criterion.Type == FieldQuery {
			fieldQueryLeftOperands[criterion.LeftOp] = true
		}
	}

	for _, newCriterion := range c2 {
		if _, ok := fieldQueryLeftOperands[newCriterion.LeftOp]; ok && newCriterion.Type == FieldQuery {
			return nil, &UnsupportedQueryError{Message: fmt.Sprintf("duplicate query key: %s", newCriterion.LeftOp)}
		}
		if err := newCriterion.Validate(); err != nil {
			return nil, err
		}
	}
	result = append(result, c2...)
	return result, nil
}

type criteriaCtxKey struct{}

// AddCriteria adds the given criteria to the context and returns an error if any of the criteria is not valid
func AddCriteria(ctx context.Context, newCriteria ...Criterion) (context.Context, error) {
	currentCriteria := CriteriaForContext(ctx)
	criteria, err := mergeCriteria(currentCriteria, newCriteria)
	if err != nil {
		return nil, err
	}
	return context.WithValue(ctx, criteriaCtxKey{}, criteria), nil
}

// CriteriaForContext returns the criteria for the given context
func CriteriaForContext(ctx context.Context) []Criterion {
	currentCriteria := ctx.Value(criteriaCtxKey{})
	if currentCriteria == nil {
		return []Criterion{}
	}
	return currentCriteria.([]Criterion)
}

// BuildCriteriaFromRequest builds criteria for the given request's query params and returns an error if the query is not valid
func BuildCriteriaFromRequest(request *web.Request) ([]Criterion, error) {
	var criteria []Criterion
	for _, queryType := range supportedQueryTypes {
		queryValues := request.URL.Query().Get(string(queryType))
		querySegments, err := process(queryValues, queryType)
		if err != nil {
			return nil, err
		}
		if criteria, err = mergeCriteria(criteria, querySegments); err != nil {
			return nil, err
		}
	}
	return criteria, nil
}

var (
	escapedOpenBracket      = url.QueryEscape(OpenBracket)
	escapedCloseBracket     = url.QueryEscape(CloseBracket)
	escapedSeparator        = url.QueryEscape(string(Separator))
	escapedOperandSeparator = url.QueryEscape(OperandSeparator)
	escapeReplacer          = strings.NewReplacer(escapedOpenBracket, OpenBracket, escapedCloseBracket, CloseBracket, escapedSeparator, string(Separator), escapedOperandSeparator, OperandSeparator, "%3D", "=")
)

// reencoding does not seem to solve any issue apart from whitespace in rightop which can be simpler solved by identifying when you';ve found both leftoperand and op
// scan symbol by symbol

// write in a buffer until you identify a known operator surrounded with +/spaces, set operator=op
// if the operator is identified, set the leftop to equal the buffer and reset the buffer
// continue reading symbols and enter an if that checks if both leftop and operator are set
// now since this is the case swich over the operator type and in the prev if (left >0 op >0) place a if separator
// 	in it if its multivalue op, read until the buffer starts bracket and ends with bracket and a separator, now set rightop, create cirterion and reset leftop,operator,rightop,buffer buffers (allows brackets and special symbols in rightop)
//  if singlevalue create criteria and reset buffers

// limitations so far:
// leftoperands cannot contain +operator+ or spaceoperatorspace for any operator
// rightoperands cannot contain ,
// but everything else would be possible - ex leftop containing + or space or , or ,, or a bracket
// also right op can contain a bracket + or space

// now in order to allow , in the right part and operators in the left part, we can introduce escaping and unescaping
// similar to this https://github.com/kubernetes/apimachinery/blob/9c4c366543346abeca2a5cd2c40cf1a30d19a2ec/pkg/fields/selector.go#L283:6
// leveraging the slash escaping , parsing the query can be simplified by first splitting by , like https://github.com/kubernetes/apimachinery/blob/9c4c366543346abeca2a5cd2c40cf1a30d19a2ec/pkg/fields/selector.go#L377:6
// and now in the parse method it wont be necessary to look for , and you would be parsing a single leftop-op-rightop criterion in a loop

func process(input string, criteriaType CriterionType) ([]Criterion, error) {
	var c []Criterion
	if input == "" {
		return c, nil
	}
	//TODO how to test corner cases
	// this breaks some queries such as ! in query
	//escapedInput := url.QueryEscape(input)
	//escapedInput = escapeReplacer.Replace(escapedInput)
	var leftOp string
	var operator Operator
	var buffer strings.Builder
	var newCriterion Criterion
	for _, ch := range input {
		// if leftop and operator are both calcualted maybe leave the rest  until a separator for rightop? what if right op has a separator though
		// the test with !
		if ch == ' ' || ch == '+' {
			if len(leftOp) > 0 {
				// we've read the left op, this must be the second + (after the operator)
				op, err := getOperator(buffer.String())
				if err != nil {
					return nil, err
				}
				operator = op
				buffer.Reset()
			} else {
				leftOp = buffer.String()
				buffer.Reset()
			}
			continue
		}
		// what if separator is in leftoperand
		if ch == Separator {
			var err error
			bufferContent := buffer.String()
			var isCriterionCandidate bool
			if operator.IsMultiVariate() {
				if strings.HasPrefix(bufferContent, OpenBracket) && strings.HasSuffix(bufferContent, CloseBracket) {
					isCriterionCandidate = true
				}
			} else {
				isCriterionCandidate = true
			}
			if isCriterionCandidate {
				newCriterion, err = convertToCriterion(leftOp, operator, bufferContent, criteriaType)
				if err != nil {
					return nil, err
				}
				leftOp = ""
				buffer.Reset()
				c = append(c, newCriterion)
				continue
			}
		}
		if _, err := buffer.WriteRune(ch); err != nil {
			return nil, err
		}
	}
	newCriterion, err := convertToCriterion(leftOp, operator, buffer.String(), criteriaType)
	if err != nil {
		return nil, err
	}
	c = append(c, newCriterion)
	return c, nil
}

// add some tests with more fancy leftop , rightops that contain special symbols (such as !@{) and also some containing
// a more complex mixed input (like rightop= this is a mixed, input example. It contains symbols + words ! -h@ppy p@rs|ng
// also how about a test with rightop json?
func convertToCriterion(leftOp string, operator Operator, rightOp string, criterionType CriterionType) (Criterion, error) {
	parsedRightOp := parseRightOp(rightOp)
	if operator == "" {
		return Criterion{}, &UnsupportedQueryError{"missing query operator"}
	}
	return newCriterion(leftOp, operator, parsedRightOp, criterionType), nil
}

func parseRightOp(rightOp string) []string {
	if strings.HasPrefix(rightOp, OpenBracket) && strings.HasSuffix(rightOp, CloseBracket) {
		rightOp = rightOp[1 : len(rightOp)-1]
		return strings.Split(rightOp, ",")
	}
	return []string{rightOp}
}

func getOperator(rawOperator string) (Operator, error) {
	for _, op := range operators {
		if string(op) == rawOperator {
			return op, nil
		}
	}
	return "", &UnsupportedQueryError{fmt.Sprintf("unsupported or missing query operator: %s", rawOperator)}
}

func isNumeric(str string) bool {
	_, err := strconv.Atoi(str)
	if err == nil {
		return true
	}
	_, err = strconv.ParseFloat(str, 64)
	return err == nil
}
