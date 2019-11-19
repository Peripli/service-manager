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

// Package logging contains log utilities and hooks for extending and customizing the logging's behaviour
package logging

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	errorSourceField = "errorSource"
)

// ErrorLocationHook provides an implementation of the sirupsen/logrus/Hook interface.
// Attaches error location information to log entries if an error is being logged and it has stack-trace information
// (i.e. if it originates from or is wrapped by github.com/pkg/errors).
type ErrorLocationHook struct {
}

// Levels implements sirupsen/logrus/Hook.Levels. The hook is fired when logging on the levels specified.
func (h *ErrorLocationHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire implements siprupsen/logrus/Hook.Fire. When fired it includes error source information in the
// log entry.
func (h *ErrorLocationHook) Fire(entry *logrus.Entry) error {
	var (
		errObj interface{}
		exists bool
	)

	if errObj, exists = entry.Data[logrus.ErrorKey]; !exists {
		return nil
	}

	err, ok := errObj.(error)
	if !ok {
		return errors.New("object logged as error does not satisfy error interface")
	}

	stackErr := getInnermostTrace(err)

	if stackErr != nil {
		stackTrace := stackErr.StackTrace()
		errSource := fmt.Sprintf("%s:%n:%d", stackTrace[0], stackTrace[0], stackTrace[0])

		entry.Data[errorSourceField] = errSource
	}

	return nil
}

type stackTracer interface {
	error
	StackTrace() errors.StackTrace
}

type causer interface {
	Cause() error
}

func getInnermostTrace(err error) stackTracer {
	var tracer stackTracer

	for {
		t, isTracer := err.(stackTracer)
		if isTracer {
			tracer = t
		}

		c, isCauser := err.(causer)
		if isCauser {
			err = c.Cause()
		} else {
			return tracer
		}
	}
}
