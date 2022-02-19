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

package log

import (
	"bytes"
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("kibana formatter", func() {

	var buffer *bytes.Buffer
	var entry *logrus.Entry

	BeforeEach(func() {
		buffer = &bytes.Buffer{}
		ctx, err := Configure(context.TODO(), &Settings{
			Level:  "debug",
			Format: "kibana",
			Output: os.Stdout.Name(),
		})
		Expect(err).ToNot(HaveOccurred())
		entry = ForContext(ctx)
		entry.Logger.SetOutput(buffer)
	})

	When("format is kibana", func() {
		JustBeforeEach(func() {
			entry.Debug("test")
		})

		It("should contain correlation_id", func() {
			Expect(buffer.String()).To(ContainSubstring(`"correlation_id":`))
		})
		It("should contain component_type", func() {
			Expect(buffer.String()).To(ContainSubstring(`"component_type":`))
		})
		It("should contain log level", func() {
			Expect(buffer.String()).To(ContainSubstring(`"level":"debug"`))
		})
		It("should contain logger information", func() {
			Expect(buffer.String()).To(ContainSubstring(`"logger":`))
		})
		It("should contain log type", func() {
			Expect(buffer.String()).To(ContainSubstring(`"type":"log"`))
		})
		It("should contain written_at human readable timestamp", func() {
			Expect(buffer.String()).To(ContainSubstring(`"written_at":`))
		})
		It("should contain written_ts timestamp", func() {
			Expect(buffer.String()).To(ContainSubstring(`"written_ts":`))
		})
		It("should contain the message", func() {
			Expect(buffer.String()).To(ContainSubstring(`"msg":"test"`))
		})
	})

	When("error is logged", func() {
		It("should append it to the message", func() {
			err := fmt.Errorf("error message")
			entry.WithError(err).Error("test message")

			Expect(buffer.String()).To(ContainSubstring(`"level":"error"`))
			Expect(buffer.String()).To(ContainSubstring(`"msg":"test message: ` + err.Error() + `"`))
		})
	})

	When("custom field is logged", func() {
		It("should not be nested", func() {
			entry.WithField("test_field", "test_value").Debug("test")

			Expect(buffer.String()).To(ContainSubstring(`,"test_field":"test_value",`))
		})
	})

})
