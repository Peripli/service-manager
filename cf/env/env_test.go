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

package env

import (
	"errors"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CF Env Suite")
}

var _ = Describe("CF Env", func() {

	BeforeSuite(func() {
		os.Setenv("VCAP_APPLICATION", "{}")
		os.Unsetenv("VCAP_SERVICES")
	})

	AfterEach(func() {
		os.Unsetenv("VCAP_SERVICES")
	})

	AfterSuite(func() {
		os.Unsetenv("VCAP_APPLICATION")
	})

	Describe("Get", func() {
		Context("existing environment variable", func() {
			It("succeeds", func() {
				testEnv := &customEnvOk{}
				os.Setenv("VCAP_SERVICES", "{}")
				os.Setenv("EXPECTED_ENV_VAR", "expected_value")
				actualValue := New(testEnv).Get("EXPECTED_ENV_VAR")
				Expect(actualValue).To(Equal("expected_value"))
			})
		})

		Context("when non cf environment variable exists", func() {
			It("should delegate the call", func() {
				testEnv := &customEnvOk{}
				os.Setenv("VCAP_SERVICES", "{}")
				actualValue := New(testEnv).Get("MISSING_ENV_VAR")
				Expect(actualValue).To(Equal("expected"))
			})
		})

	})

	Describe("Load", func() {

		Context("with valid postgresql service", func() {
			It("succeeds", func() {
				testEnv := &customEnvOk{}
				vcapServices := `{ "postgresql": [{
					"credentials": { "uri": "expectedUri" },
					"name": "postgresql"
				}]}`
				os.Setenv("VCAP_SERVICES", vcapServices)
				err := New(testEnv).Load()
				Expect(err).To(BeNil())
			})
		})

		Context("with missing postgresql service", func() {
			It("returns error", func() {
				testEnv := &customEnvOk{}
				vcapServices := `{ "notpgservice": [{"credentials": {}}] }`
				os.Setenv("VCAP_SERVICES", vcapServices)
				err := New(testEnv).Load()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("Could not find service with name postgresql"))
			})
		})

		Context("with failing delegate", func() {
			It("returns error", func() {
				testEnv := &customEnvFail{}
				err := New(testEnv).Load()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("expected fail"))
			})
		})

		Context("with invalid VCAP_SERVICES", func() {
			It("returns error", func() {
				os.Setenv("VCAP_SERVICES", "Invalid")
				testEnv := &customEnvOk{}
				Expect(New(testEnv).Load()).To(HaveOccurred())
			})
		})

		Context("with missing VCAP_SERVICES", func() {
			It("returns error", func() {
				testEnv := &customEnvOk{}
				Expect(New(testEnv).Load()).To(HaveOccurred())
			})
		})
	})
})

type customEnvOk struct{}

func (env *customEnvOk) Load() error { return nil }
func (env *customEnvOk) Get(key string) interface{} {
	if key == "db.name" {
		return "postgresql"
	}
	return "expected"
}
func (env *customEnvOk) Set(key string, value interface{}) {
	Expect(key).To(Equal("db.uri"))
	Expect(value.(string)).To(Equal("expectedUri"))
}
func (env *customEnvOk) Unmarshal(value interface{}) error { return nil }

type customEnvFail struct{}

func (env *customEnvFail) Load() error                       { return errors.New("expected fail") }
func (env *customEnvFail) Get(key string) interface{}        { return "expected" }
func (env *customEnvFail) Set(key string, value interface{}) {}
func (env *customEnvFail) Unmarshal(value interface{}) error { return errors.New("expected fail") }
