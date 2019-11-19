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

package logging

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

var _ = Describe("ErrorLocationHook", func() {

	Describe("Levels", func() {
		It("matches all levels", func() {
			hook := &ErrorLocationHook{}
			Expect(hook.Levels()).To(Equal(logrus.AllLevels))
		})
	})

	Describe("Fire", func() {
		formatErrorSource := func(tracer interface{}) string {
			stackTracer, ok := tracer.(stackTracer)
			if !ok {
				return ""
			}
			stackTrace := stackTracer.StackTrace()

			return fmt.Sprintf("%s:%n:%d", stackTrace[0], stackTrace[0], stackTrace[0])
		}

		prepError := func(isPkgErr bool, stackCount, wrapCount int) interface{} {
			var err error
			if isPkgErr {
				err = errors.New("pkg error")
			} else {
				err = fmt.Errorf("normal error")
			}

			for i := 1; i <= wrapCount; i++ {
				err = errors.Wrap(err, "wrapMessage")
			}

			for i := 1; i <= stackCount; i++ {
				err = errors.WithStack(err)
			}

			return err
		}

		type testCase struct {
			actualErrorValue interface{}
			stackTimes       int
			wrapTimes        int

			expectedFireError        error
			expectedErrorSourceValue string
		}

		entries := []TableEntry{
			Entry("When error key field is missing it returns nil", testCase{
				actualErrorValue:         nil,
				expectedFireError:        nil,
				expectedErrorSourceValue: "",
			}),

			Entry("When error key field is not an error it returns an error", testCase{
				actualErrorValue: func(err error, stacks, wraps int) interface{} {
					return struct{ A string }{A: "val"}
				},
				expectedFireError:        fmt.Errorf("object logged as error does not satisfy error interface"),
				expectedErrorSourceValue: "",
			}),

			Entry("When error key field contains a normal error it returns nil", testCase{
				actualErrorValue:         prepError(false, 0, 0),
				expectedFireError:        nil,
				expectedErrorSourceValue: "",
			}),

			Entry("When error key field contains a n-wrapped normal error it returns stack from the inner-most causer after n-1 unwraps", testCase{
				actualErrorValue:         prepError(false, 0, 20),
				expectedFireError:        nil,
				expectedErrorSourceValue: formatErrorSource(prepError(false, 0, 1)),
			}),
			Entry("When error key field contains a n-stacked normal error it returns stack from the inner-most causer after n-1 unwraps", testCase{
				actualErrorValue:         prepError(false, 20, 0),
				expectedFireError:        nil,
				expectedErrorSourceValue: formatErrorSource(prepError(false, 1, 0)),
			}),

			Entry("When error key field contains a pkg/error it returns it", testCase{
				actualErrorValue:         prepError(true, 0, 0),
				expectedFireError:        nil,
				expectedErrorSourceValue: formatErrorSource(prepError(true, 0, 0)),
			}),

			Entry("When error key field contains a n-wrapped pkg/error it returns stack from the inner-most causer after n unwraps", testCase{
				actualErrorValue:         prepError(true, 20, 0),
				expectedFireError:        nil,
				expectedErrorSourceValue: formatErrorSource(prepError(true, 0, 0)),
			}),
			Entry("When error key field contains a n-stacked pkg/error it returns stack from the inner-most causer after n unwraps", testCase{
				actualErrorValue:         prepError(true, 0, 20),
				expectedFireError:        nil,
				expectedErrorSourceValue: formatErrorSource(prepError(true, 0, 0)),
			}),
		}

		DescribeTable("Fire Hook", func(t testCase) {
			hook := &ErrorLocationHook{}
			entry := logrus.NewEntry(logrus.New())

			if t.actualErrorValue != nil {
				entry.Data[logrus.ErrorKey] = t.actualErrorValue
			}

			err := hook.Fire(entry)
			if t.expectedFireError != nil {
				Expect(err.Error()).To(Equal(t.expectedFireError.Error()))
			} else {
				Expect(err).ShouldNot(HaveOccurred())
			}
			Expect(cast.ToString(entry.Data[errorSourceField])).To(Equal(t.expectedErrorSourceValue))

		}, entries...)
	})
})
