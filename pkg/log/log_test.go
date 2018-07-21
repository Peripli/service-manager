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

// Package log contains logic for setting up logging for SM
package log

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

// TestServiceManager tests servermanager package
func TestLog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Log Suite")
}

var _ = Describe("log", func() {
	Describe("SetupLogging", func() {

		Context("with invalid log level", func() {
			It("should panic", func() {
				expectPanic(Settings{
					Level:  "invalid",
					Format: "text",
				})
			})
		})

		Context("with invalid log level", func() {
			It("should panic", func() {
				expectPanic(Settings{
					Level:  "debug",
					Format: "invalid",
				})
			})
		})

		Context("with text log level", func() {
			It("should log text", func() {
				expectOutput("msg=Test", "text")
			})
		})

		Context("with json log level", func() {
			It("should log json", func() {
				expectOutput("\"msg\":\"Test\"", "json")
			})
		})

	})
})

type MyWriter struct {
	Data string
}

func (wr *MyWriter) Write(p []byte) (n int, err error) {
	wr.Data += string(p)
	return len(p), nil
}

func expectPanic(settings Settings) {
	wrapper := func() {
		SetupLogging(settings)
	}
	Expect(wrapper).To(Panic())
}

func expectOutput(substring string, logFormat string) {
	w := &MyWriter{}
	SetupLogging(Settings{
		Level:  "debug",
		Format: logFormat,
	})
	logrus.SetOutput(w)
	defer logrus.SetOutput(os.Stderr) // return default output
	logrus.Debug("Test")
	Expect(w.Data).To(ContainSubstring(substring))
}
