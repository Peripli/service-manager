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

package env_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"io/ioutil"
	"os"
	"testing"

	"github.com/Peripli/service-manager/env"
	"github.com/Peripli/service-manager/server"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Env Suite")
}

var (
	yamlExample = []byte(`debug: true
port: 8080
database:
  uri: localhost:8080
`)
	partialYamlExample = []byte(`
database:
  uri: localhost:8080
`)
	expected = map[string]interface{}{
		"debug": true,
		"port":  8080,
		"database": map[string]interface{}{
			"uri": "localhost:8080",
		},
	}
)

type environment struct {
	Debug    bool
	Port     int
	Database dbSettings
}

type dbSettings struct {
	URI string
}

var _ = Describe("Env", func() {
	var defaultEnv server.Environment
	loadEnvironmentFromFile := func(yaml []byte) error {
		err := ioutil.WriteFile("application.yml", yaml, 0640)
		Expect(err).To(BeNil())
		defer func() {
			os.Remove("application.yml")
		}()
		return defaultEnv.Load()
	}

	BeforeEach(func() {
		defaultEnv = env.New(&env.ConfigFile{
			Path:   ".",
			Name:   "application",
			Format: "yml",
		}, "SM")
	})

	Describe("Load environment", func() {
		Context("When file is read successfully", func() {
			It("Should return nil", func() {
				Expect(loadEnvironmentFromFile(yamlExample)).To(BeNil())
			})
		})

		Context("When file exists but contains only mandatory properties", func() {
			It("Should return nil", func() {
				Expect(loadEnvironmentFromFile(partialYamlExample)).To(BeNil())
			})
		})
	})

	Describe("Get property", func() {
		It("Should return one from loaded configuration", func() {
			Expect(loadEnvironmentFromFile(yamlExample)).To(BeNil())
			for key, expectedValue := range expected {
				Expect(defaultEnv.Get(key)).To(Equal(expectedValue))
			}
		})
	})

	Describe("Set Property", func() {
		It("Should put it in the environment", func() {
			Expect(loadEnvironmentFromFile(yamlExample)).To(BeNil())
			defaultEnv.Set("some.key", "some.value")
			Expect(defaultEnv.Get("some.key")).To(Equal("some.value"))
		})

		It("Should override existing value for key", func() {
			Expect(loadEnvironmentFromFile(yamlExample)).To(BeNil())
			defaultEnv.Set("port", "1234")
			actual := defaultEnv.Get("port")
			Expect(actual).To(Not(Equal(expected["port"])))
			Expect(actual).To(Equal("1234"))
		})
	})

	Describe("Unmarshal", func() {
		BeforeEach(func() {
			Expect(loadEnvironmentFromFile(yamlExample)).To(BeNil())
		})

		Context("With non-struct parameter", func() {
			It("Should return an error", func() {
				Expect(defaultEnv.Unmarshal(10)).To(Not(BeNil()))
			})
		})
		Context("With non-pointer-struct parameter", func() {
			It("Should return an error", func() {
				Expect(defaultEnv.Unmarshal(environment{})).To(Not(BeNil()))
			})
		})
		Context("With struct parameter", func() {
			It("Should unmarshal correctly", func() {
				envToLoad := &environment{}
				Expect(defaultEnv.Unmarshal(envToLoad)).To(BeNil())
				Expect(envToLoad.Debug).To(Equal(expected["debug"]))
				Expect(envToLoad.Port).To(Equal(expected["port"]))
				Expect(envToLoad.Database.URI).To(Equal(expected["database"].(map[string]interface{})["uri"]))
			})
		})
	})
})
