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

package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"io/ioutil"
	"os"
	"testing"

	"path/filepath"

	"strings"

	"github.com/Peripli/service-manager/config"
	"github.com/fatih/structs"
	"github.com/spf13/cast"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Env Suite")
}

var _ = Describe("Env", func() {
	const (
		key              = "test"
		flagDefaultValue = "pflagDefaultValue"
		fileValue        = "fileValue"
		envValue         = "envValue"
		flagValue        = "pflagValue"
		overrideValue    = "overrideValue"

		keyWbool      = "wbool"
		keyWint       = "wint"
		keyWstring    = "wstring"
		keyWmappedVal = "w_mapped_val"
		keyNbool      = "nest_nbool"

		description   = "desc"
		keyNint       = "nest_nint"
		keyNstring    = "nest_nstring"
		keyNmappedVal = "nest_n_mapped_val"
	)
	type Nest struct {
		NBool      bool
		NInt       int
		NString    string
		NMappedVal string `mapstructure:"n_mapped_val" structs:"n_mapped_val"  yaml:"n_mapped_val"`
	}

	type Outer struct {
		WBool      bool
		WInt       int
		WString    string
		WMappedVal string `mapstructure:"w_mapped_val" structs:"w_mapped_val" yaml:"w_mapped_val"`
		Nest       Nest
	}

	var (
		properties        map[string]interface{}
		structure         Outer
		overrideStructure Outer
		env               config.Environment
		file              config.File
	)

	addStructurePFlags := func() *pflag.FlagSet {
		set := pflag.NewFlagSet("testflags", pflag.ExitOnError)

		set.Bool(keyWbool, structure.WBool, description)
		set.Int(keyWint, structure.WInt, description)
		set.String(keyWstring, structure.WString, description)
		set.String(keyWmappedVal, structure.WMappedVal, description)

		set.Bool(keyNbool, structure.Nest.NBool, description)
		set.Int(keyNint, structure.Nest.NInt, description)
		set.String(keyNstring, structure.Nest.NString, description)
		set.String(keyNmappedVal, structure.Nest.NMappedVal, description)

		pflag.CommandLine.AddFlagSet(set)

		return set
	}

	addSingleFlag := func(key, defaultValue, description string) *pflag.FlagSet {
		set := pflag.NewFlagSet("testflags", pflag.ExitOnError)
		set.String(key, defaultValue, description)

		pflag.CommandLine.AddFlagSet(set)

		return set
	}

	setPFlags := func() {
		Expect(pflag.Set(keyWbool, cast.ToString(overrideStructure.WBool))).ShouldNot(HaveOccurred())
		Expect(pflag.Set(keyWint, cast.ToString(overrideStructure.WInt))).ShouldNot(HaveOccurred())
		Expect(pflag.Set(keyWstring, overrideStructure.WString)).ShouldNot(HaveOccurred())
		Expect(pflag.Set(keyWmappedVal, overrideStructure.WMappedVal)).ShouldNot(HaveOccurred())

		Expect(pflag.Set(keyNbool, cast.ToString(overrideStructure.Nest.NBool))).ShouldNot(HaveOccurred())
		Expect(pflag.Set(keyNint, cast.ToString(overrideStructure.Nest.NInt))).ShouldNot(HaveOccurred())
		Expect(pflag.Set(keyNstring, overrideStructure.Nest.NString)).ShouldNot(HaveOccurred())
		Expect(pflag.Set(keyNmappedVal, overrideStructure.Nest.NMappedVal)).ShouldNot(HaveOccurred())
	}

	setEnvVars := func() {
		Expect(os.Setenv(strings.ToTitle(keyWbool), cast.ToString(structure.WBool))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyWint), cast.ToString(structure.WInt))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyWstring), structure.WString)).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyWmappedVal), structure.WMappedVal)).ShouldNot(HaveOccurred())

		Expect(os.Setenv(strings.ToTitle(keyNbool), cast.ToString(structure.Nest.NBool))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyNint), cast.ToString(structure.Nest.NInt))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyNstring), structure.Nest.NString)).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyNmappedVal), structure.Nest.NMappedVal)).ShouldNot(HaveOccurred())
	}

	unsetEnvVars := func() {
		Expect(os.Unsetenv(keyWbool)).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(keyWint)).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(keyWstring)).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(keyWmappedVal)).ShouldNot(HaveOccurred())

		Expect(os.Unsetenv(keyNbool)).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(keyNint)).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(keyNstring)).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(keyNmappedVal)).ShouldNot(HaveOccurred())

		Expect(os.Unsetenv(key)).ShouldNot(HaveOccurred())
	}

	loadEnv := func() {
		err := ioutil.WriteFile("application.yml", []byte{}, 0640)
		defer func() {
			Expect(os.Remove("application.yml")).ShouldNot(HaveOccurred())

		}()
		Expect(ioutil.WriteFile("application.yml", []byte{}, 0640)).ShouldNot(HaveOccurred())

		err = env.Load()
		Expect(err).ShouldNot(HaveOccurred())
	}

	loadEnvWithConfigFile := func(file config.File, content interface{}) {
		if content != nil {
			f := file.Location + string(filepath.Separator) + file.Name + "." + file.Format
			bytes, err := yaml.Marshal(content)
			Expect(err).ShouldNot(HaveOccurred())
			err = ioutil.WriteFile(f, bytes, 0640)
			Expect(err).ShouldNot(HaveOccurred())

			defer func() {
				os.Remove(f)
			}()
		}

		err := env.Load()
		Expect(err).ShouldNot(HaveOccurred())
	}

	verifyEnvContainsValues := func(expected interface{}) {
		props := structs.Map(expected)
		for key, expectedValue := range props {
			switch v := expectedValue.(type) {
			case map[string]interface{}:
				for nestedKey, nestedExpectedValue := range v {
					Expect(cast.ToString(env.Get(key + "_" + nestedKey))).Should(Equal(cast.ToString(nestedExpectedValue)))
				}
			default:
				Expect(cast.ToString(env.Get(key))).To(Equal(cast.ToString(expectedValue)))
			}
		}
	}

	BeforeEach(func() {
		file = config.DefaultFile()
		env = config.NewEnv()
		Expect(env).ShouldNot(BeNil())

		structure = Outer{
			WBool:      true,
			WInt:       1234,
			WString:    "wstringval",
			WMappedVal: "wmappedval",
			Nest: Nest{
				NBool:      true,
				NInt:       4321,
				NString:    "nstringval",
				NMappedVal: "nmappedval",
			},
		}

		overrideStructure = Outer{
			WBool:      false,
			WInt:       8888,
			WString:    "overrideval",
			WMappedVal: "overrideval",
			Nest: Nest{
				NBool:      false,
				NInt:       9999,
				NString:    "overrideval",
				NMappedVal: "overrideval",
			},
		}
	})

	AfterEach(func() {
		// clean up env
		unsetEnvVars()

		// clean up pflags
		pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	})

	Describe("Load", func() {
		const (
			keyFileName     = "file_name"
			keyFileLocation = "file_location"
			keyFileFormat   = "file_format"
		)

		It("adds default viper bindings to standard pflags", func() {
			addStructurePFlags()

			loadEnv()
			verifyEnvContainsValues(structure)
		})

		Context("when SM config file doesn't exist", func() {
			It("returns an error", func() {
				Expect(env.Load()).Should(HaveOccurred())
			})
		})

		Context("when SM config file exists", func() {
			It("creates pflags for SM config file with proper default values", func() {
				loadEnvWithConfigFile(file, structure)

				Expect(pflag.Lookup("file_name").Value.String()).Should(Equal(file.Name))
				Expect(pflag.Lookup("file_location").Value.String()).Should(Equal(file.Location))
				Expect(pflag.Lookup("file_format").Value.String()).Should(Equal(file.Format))
			})

			It("binds the SM config file pflags to the env", func() {
				loadEnvWithConfigFile(file, structure)

				Expect(env.Get(keyFileName)).Should(Equal(file.Name))
				Expect(env.Get(keyFileLocation)).Should(Equal(file.Location))
				Expect(env.Get(keyFileFormat)).Should(Equal(file.Format))

				cfgFile := struct {
					File config.File
				}{}

				Expect(env.Unmarshal(&cfgFile)).ShouldNot(HaveOccurred())
				Expect(cfgFile.File).Should(Equal(file))
			})

			It("allows overriding the config file properties", func() {
				file.Name = "updatedName"
				pflag.String(keyFileName, "application", "")
				pflag.Set(keyFileName, "updatedName")

				loadEnvWithConfigFile(file, structure)

				verifyEnvContainsValues(structure)
			})
		})
	})

	Describe("BindPFlag", func() {
		const (
			key         = "test_flag"
			description = description
			aliasKey    = "test.flag"
		)
		It("allows getting a pflag from the env with an alias name", func() {
			addSingleFlag(key, flagDefaultValue, description)

			env.Load()

			Expect(env.Get(key)).To(Equal(flagDefaultValue))
			Expect(env.Get(aliasKey)).To(BeNil())

			env.BindPFlag(aliasKey, pflag.Lookup(key))

			Expect(env.Get(key)).To(Equal(flagDefaultValue))
			Expect(env.Get(aliasKey)).To(Equal(flagDefaultValue))
		})
	})

	Describe("Get", func() {
		Context("when properties are loaded via standard pflags", func() {
			BeforeEach(func() {
				addStructurePFlags()
			})

			It("returns the default flag value if the flag is not set", func() {
				loadEnv()

				verifyEnvContainsValues(structure)
			})

			It("returns the flags values if the flags are set", func() {
				setPFlags()
				loadEnv()
				verifyEnvContainsValues(overrideStructure)

			})
		})

		Context("when properties are loaded via pflags structure binding", func() {
			BeforeEach(func() {
				env.CreatePFlags(structure)
			})

			It("returns the default flag value if the flag is not set", func() {
				loadEnv()

				verifyEnvContainsValues(structure)
			})

			It("returns the flags values if the flags are set", func() {
				setPFlags()
				loadEnv()

				verifyEnvContainsValues(overrideStructure)
			})
		})

		Context("when properties are loaded via config file", func() {
			It("returns values from config file", func() {
				loadEnvWithConfigFile(file, structure)

				verifyEnvContainsValues(structure)
			})
		})

		Context("when properties are loaded via OS env variable", func() {
			BeforeEach(func() {
				setEnvVars()
			})

			It("returns values from env", func() {
				loadEnv()

				verifyEnvContainsValues(structure)
			})

		})

		Context("override > pflag set > env > file > pflag default", func() {
			BeforeEach(func() {
				properties = map[string]interface{}{
					key: fileValue,
				}
			})

			It("respects loading order", func() {
				addSingleFlag(key, flagDefaultValue, "")
				loadEnv()
				Expect(env.Get(key)).Should(Equal(flagDefaultValue))

				loadEnvWithConfigFile(file, properties)
				Expect(env.Get(key)).Should(Equal(fileValue))

				os.Setenv(strings.ToTitle(key), envValue)
				Expect(env.Get(key)).Should(Equal(envValue))

				pflag.Set(key, flagValue)
				Expect(env.Get(key)).Should(Equal(flagValue))

				env.Set(key, overrideValue)
				Expect(env.Get(key)).Should(Equal(overrideValue))
			})
		})
	})

	Describe("Set", func() {

		It("creates an alias for the value in case it contains dot separators", func() {
			loadEnv()
			env.Set("test.key", "value")

			Expect(env.Get("test_key")).To(Equal("value"))
		})

		It("adds the property in the environment abstraction", func() {
			loadEnv()
			env.Set(key, overrideValue)

			Expect(env.Get(key)).To(Equal(overrideValue))
		})

		It("has highest priority", func() {
			properties = map[string]interface{}{
				key: fileValue,
			}
			addSingleFlag(key, flagDefaultValue, "")
			os.Setenv(key, envValue)
			loadEnvWithConfigFile(file, properties)

			env.Set(key, overrideValue)
			pflag.Set(key, flagValue)

			Expect(env.Get(key)).Should(Equal(overrideValue))
		})
	})

	Describe("Unmarshal", func() {
		var actual Outer

		BeforeEach(func() {
			actual = Outer{}
		})

		Context("when parameter is not a struct", func() {
			It("returns an error", func() {
				loadEnv()

				Expect(env.Unmarshal(10)).To(Not(BeNil()))
			})
		})

		Context("when parameter is not a pointer to a struct", func() {
			It("returns an error", func() {
				loadEnv()

				Expect(env.Unmarshal(actual)).To(Not(BeNil()))
			})
		})

		Context("parameter is a pointer to a struct", func() {
			verifyUnmarshallingIsCorrect := func(actual, expected interface{}) {
				err := env.Unmarshal(actual)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(actual).To(Equal(expected))
			}

			Context("when properties are loaded via standard pflags", func() {
				BeforeEach(func() {
					addStructurePFlags()

				})

				It("unmarshals correctly", func() {
					loadEnv()

					verifyUnmarshallingIsCorrect(&actual, &structure)
				})
			})

			Context("when properties are loaded via pflags structure binding", func() {
				BeforeEach(func() {
					env.CreatePFlags(structure)
				})

				It("unmarshals correctly", func() {
					loadEnv()

					verifyUnmarshallingIsCorrect(&actual, &structure)
				})
			})

			Context("when property is loaded via config file", func() {
				It("unmarshals correctly", func() {
					loadEnvWithConfigFile(file, structure)

					verifyUnmarshallingIsCorrect(&actual, &structure)
				})
			})

			Context("when properties are loaded via OS env variable", func() {
				BeforeEach(func() {
					setEnvVars()
				})

				It("unmarshals correctly", func() {
					loadEnv()

					verifyUnmarshallingIsCorrect(&actual, &structure)
				})
			})

			Context("override > pflag set > env > file > pflag default", func() {
				type s struct {
					Test string
				}

				var str s

				BeforeEach(func() {
					str = s{}
				})

				It("respects loading order", func() {
					addSingleFlag(key, flagDefaultValue, "")
					loadEnv()
					verifyUnmarshallingIsCorrect(&str, &s{flagDefaultValue})

					loadEnvWithConfigFile(file, s{fileValue})
					verifyUnmarshallingIsCorrect(&str, &s{fileValue})

					os.Setenv(strings.ToTitle(key), envValue)
					verifyUnmarshallingIsCorrect(&str, &s{envValue})
					Expect(env.Get(key)).Should(Equal(envValue))

					pflag.Set(key, flagValue)
					verifyUnmarshallingIsCorrect(&str, &s{flagValue})

					env.Set(key, overrideValue)
					verifyUnmarshallingIsCorrect(&str, &s{overrideValue})
				})
			})
		})
	})
})
