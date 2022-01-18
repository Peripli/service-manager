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
	"bytes"
	"context"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

// TestLog tests logging package
func TestLog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Log Suite")
}

// TestMultipleGoroutinesDefaultLog validates that no race conditions occur when two go routines log using the default log
func TestMultipleGoroutinesDefaultLog(t *testing.T) {
	_, err := Configure(context.TODO(), DefaultSettings())
	Expect(err).ToNot(HaveOccurred())
	go func() { D().Debug("message") }()
	go func() { D().Debug("message") }()
}

// TestMultipleGoroutinesContextLog validates that no race conditions occur when two go routines log using the context log
func TestMultipleGoroutinesContextLog(t *testing.T) {
	ctx, err := Configure(context.Background(), DefaultSettings())
	Expect(err).ToNot(HaveOccurred())
	go func() { C(ctx).Debug("message") }()
	go func() { C(ctx).Debug("message") }()
}

// TestMultipleGoroutinesMixedLog validates that no race conditions occur when two go routines log using both context and default log
func TestMultipleGoroutinesMixedLog(t *testing.T) {
	ctx, err := Configure(context.TODO(), DefaultSettings())
	Expect(err).ToNot(HaveOccurred())
	go func() { C(ctx).Debug("message") }()
	go func() { D().Debug("message") }()
}

var _ = Describe("log", func() {
	Describe("SetupLogging", func() {

		Context("with invalid log level", func() {
			It("returns an error", func() {
				expectConfigModificationToFail(&Settings{
					Level:  "invalid",
					Format: "text",
					Output: os.Stderr.Name(),
				})
			})
		})

		Context("with invalid log format", func() {
			It("returns an error", func() {
				expectConfigModificationToFail(&Settings{
					Level:  "debug",
					Format: "invalid",
					Output: os.Stderr.Name(),
				})
			})
		})

		Context("with invalid log format", func() {
			It("returns an error", func() {
				expectConfigModificationToFail(&Settings{
					Level:  "debug",
					Format: "text",
					Output: "invalid",
				})
			})
		})

		Context("with text log format", func() {
			It("should log text", func() {
				expectOutput("msg=Test", "text")
			})
		})

		Context("with json log format", func() {
			It("should log json", func() {
				expectOutput("\"msg\":\"Test\"", "json")
			})
		})

		Context("with kibana log format", func() {
			It("should log in kibana format", func() {
				expectOutput(`"component_type":"application","correlation_id":"-"`, "kibana")
				expectOutput(`"msg":"Test","type":"log"`, "kibana")
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

func expectConfigModificationToFail(settings *Settings) {
	previousConfig := Configuration()
	_, err := Configure(context.TODO(), settings)
	Expect(err).To(HaveOccurred())
	currentConfig := Configuration()
	Expect(previousConfig).To(Equal(currentConfig))
}

func expectOutput(substring string, logFormat string) {
	w := &bytes.Buffer{}
	ctx, err := Configure(context.TODO(), &Settings{
		Level:  "debug",
		Format: logFormat,
		Output: os.Stderr.Name(),
	})
	Expect(err).ToNot(HaveOccurred())
	entry := ForContext(ctx)
	entry.Logger.SetOutput(w)
	defer entry.Logger.SetOutput(os.Stderr) // return default output
	entry.Debug("Test")
	Expect(w.String()).To(ContainSubstring(substring))
}
