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

package health

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Healthcheck Settings", func() {

	var settings *Settings
	var fatal bool
	var failuresTreshold int64
	var interval time.Duration

	BeforeEach(func() {
		settings = DefaultSettings()

		fatal = false
		failuresTreshold = 1
		interval = 30
	})

	var registerIndicatorSettings = func() {
		indicatorSettings := &IndicatorSettings{
			Fatal:            fatal,
			FailuresTreshold: failuresTreshold,
			Interval:         interval,
		}

		settings.IndicatorsSettings["test"] = indicatorSettings
	}

	var assertValidationErrorOccured = func(err error) {
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("validate Settings"))
	}

	When("Indicator with negative treshold", func() {
		It("Should be invalid", func() {
			failuresTreshold = -1
			registerIndicatorSettings()

			err := settings.Validate()

			assertValidationErrorOccured(err)
		})
	})

	When("Indicator with 0 treshold", func() {
		It("Should be invalid", func() {
			failuresTreshold = 0
			registerIndicatorSettings()

			err := settings.Validate()

			assertValidationErrorOccured(err)
		})
	})

	When("Indicator with interval less than 30", func() {
		It("Should be invalid", func() {
			interval = 15
			registerIndicatorSettings()

			err := settings.Validate()

			assertValidationErrorOccured(err)
		})
	})
})
