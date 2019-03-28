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
	"context"
	"fmt"
	"os"
	"sync"
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

// TestMultipleGoroutinesDefaultLog validates that no race conditions occur when two go routines log using the default log
func TestMultipleGoroutinesDefaultLog(t *testing.T) {
	Configure(context.TODO(), DefaultSettings())
	go func() { D().Debug("message") }()
	go func() { D().Debug("message") }()
}

// TestMultipleGoroutinesContextLog validates that no race conditions occur when two go routines log using the context log
func TestMultipleGoroutinesContextLog(t *testing.T) {
	ctx := Configure(context.Background(), DefaultSettings())
	go func() { C(ctx).Debug("message") }()
	go func() { C(ctx).Debug("message") }()
}

// TestMultipleGoroutinesMixedLog validates that no race conditions occur when two go routines log using both context and default log
func TestMultipleGoroutinesMixedLog(t *testing.T) {
	ctx := Configure(context.TODO(), DefaultSettings())
	go func() { C(ctx).Debug("message") }()
	go func() { D().Debug("message") }()
}

var _ = Describe("log", func() {
	Describe("SetupLogging", func() {

		Context("with invalid log level", func() {
			It("should panic", func() {
				expectPanic(&Settings{
					Level:  "invalid",
					Format: "text",
				})
			})
		})

		Context("with invalid log level", func() {
			It("should panic", func() {
				expectPanic(&Settings{
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

	Describe("Register formatter", func() {
		Context("When a formatter with such name is not registered", func() {
			It("Registers it", func() {
				name := "formatter_name"
				err := RegisterFormatter(name, &logrus.TextFormatter{})
				Expect(err).ToNot(HaveOccurred())
				Expect(supportedFormatters[name]).ToNot(BeNil())
			})
		})
		Context("When a formatter with such name is registered", func() {
			It("Returns an error", func() {
				err := RegisterFormatter("text", &logrus.TextFormatter{})
				Expect(err).To(HaveOccurred())
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

func expectPanic(settings *Settings) {
	wrapper := func() {
		configure(settings)
	}
	Expect(wrapper).To(Panic())
}

func expectOutput(substring string, logFormat string) {
	w := &MyWriter{}
	ctx := configure(&Settings{
		Level:  "debug",
		Format: logFormat,
	})
	entry := ForContext(ctx)
	entry.Logger.SetOutput(w)
	defer entry.Logger.SetOutput(os.Stderr) // return default output
	entry.Debug("Test")
	fmt.Println(w.Data)
	Expect(w.Data).To(ContainSubstring(substring))
}

func configure(settings *Settings) context.Context {
	once = sync.Once{}
	return Configure(context.TODO(), settings)
}
