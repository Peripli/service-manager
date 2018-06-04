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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "K8S Env Suite")
}

var _ = Describe("K8S Env", func() {

	BeforeSuite(func() {
		os.Unsetenv(K8SPostgresConfigLocationEnvVarName)
	})

	Describe("Load", func() {

		AfterEach(func() {
			os.Unsetenv(K8SPostgresConfigLocationEnvVarName)
		})

		Context("with missing postgresql config file location", func() {
			It("returns error", func() {
				testEnv := &customEnvOk{}
				err := New(testEnv).Load()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("Expected " + K8SPostgresConfigLocationEnvVarName + " environment variable to be set"))
			})
		})

		Context("with invalid postgresql config file location", func() {
			It("returns error", func() {
				os.Setenv(K8SPostgresConfigLocationEnvVarName, "invalid/postgresql/config/location")
				testEnv := &customEnvOk{}
				err := New(testEnv).Load()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("Could not get configuration"))
			})
		})

		Context("with empty postgresql config file", func() {
			It("returns error", func() {
				dir, err := ioutil.TempDir("", "k8s_env_test")
				Expect(err).To(BeNil())
				defer os.RemoveAll(dir)

				err = ioutil.WriteFile(filepath.Join(dir, "uri"), []byte(""), os.ModePerm)
				Expect(err).To(BeNil())

				os.Setenv(K8SPostgresConfigLocationEnvVarName, dir)
				testEnv := &customEnvOk{}
				err = New(testEnv).Load()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("Configuration for uri is empty"))
			})
		})

		Context("with expected postgresql config file", func() {
			It("succeeds", func() {
				dir, err := ioutil.TempDir("", "k8s_env_test")
				Expect(err).To(BeNil())
				defer os.RemoveAll(dir)

				err = ioutil.WriteFile(filepath.Join(dir, "uri"), []byte("expected"), os.ModePerm)
				Expect(err).To(BeNil())

				os.Setenv(K8SPostgresConfigLocationEnvVarName, dir)
				testEnv := &customEnvOk{}
				err = New(testEnv).Load()
				Expect(err).To(BeNil())
			})
		})

		Context("with failing delegate env", func() {
			It("returns error", func() {
				testEnv := &customEnvFail{}
				err := New(testEnv).Load()
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("expected"))
			})
		})

	})
})

type customEnvOk struct{}

func (env *customEnvOk) Load() error                { return nil }
func (env *customEnvOk) Get(key string) interface{} { return "expected" }
func (env *customEnvOk) Set(key string, value interface{}) {
	Expect(key).To(Equal("db.uri"))
	Expect(value.(string)).To(Equal("expected"))
}
func (env *customEnvOk) Unmarshal(value interface{}) error { return nil }

type customEnvFail struct{}

func (env *customEnvFail) Load() error                       { return errors.New("expected") }
func (env *customEnvFail) Get(key string) interface{}        { return "expected" }
func (env *customEnvFail) Set(key string, value interface{}) {}
func (env *customEnvFail) Unmarshal(value interface{}) error { return errors.New("expected") }
