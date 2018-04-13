package env_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"io/ioutil"
	"os"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/env"
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
	loadEnvironmentFromFile := func() error {
		err := ioutil.WriteFile("application.yml", yamlExample, 0640)
		Expect(err).To(BeNil())
		defer func() {
			os.Remove("application.yml")
		}()
		return defaultEnv.Load()
	}

	BeforeEach(func() {
		defaultEnv = env.Default()
	})

	Describe("Load environment", func() {
		Context("When file doesn't exist", func() {
			It("Should panic", func() {
				Expect(defaultEnv.Load()).To(Not(BeNil()))
			})
		})

		Context("When file is read successfully", func() {
			It("Should return nil", func() {
				Expect(loadEnvironmentFromFile()).To(BeNil())
			})
		})
	})

	Describe("Get property", func() {
		It("Should return one from loaded configuration", func() {
			Expect(loadEnvironmentFromFile()).To(BeNil())
			for key, expectedValue := range expected {
				Expect(defaultEnv.Get(key)).To(Equal(expectedValue))
			}
		})
	})

	Describe("Unmarshal", func() {
		BeforeEach(func() {
			Expect(loadEnvironmentFromFile()).To(BeNil())
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
