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
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"io/ioutil"
	"os"
	"testing"

	"path/filepath"

	"strings"

	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/fatih/structs"
	"github.com/spf13/cast"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

func TestEnv(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Env Suite")
}

var _ = Describe("Env", func() {
	const (
		key              = "key"
		description      = "desc"
		flagDefaultValue = "pflagDefaultValue"
		fileValue        = "fileValue"
		envValue         = "envValue"
		flagValue        = "pflagValue"
		overrideValue    = "overrideValue"

		keyWbool      = "wbool"
		keyWint       = "wint"
		keyWstring    = "wstring"
		keyWmappedVal = "w_mapped_val"

		keyNbool      = "nest.nbool"
		keyNint       = "nest.nint"
		keyNstring    = "nest.nstring"
		keyNslice     = "nest.nslice"
		keyNmappedVal = "nest.n_mapped_val"
	)
	type Nest struct {
		NBool      bool
		NInt       int
		NString    string
		NSlice     []string
		NMappedVal string `mapstructure:"n_mapped_val" structs:"n_mapped_val"  yaml:"n_mapped_val"`
	}

	type Outer struct {
		WBool      bool
		WInt       int
		WString    string
		WMappedVal string `mapstructure:"w_mapped_val" structs:"w_mapped_val" yaml:"w_mapped_val"`
		Nest       Nest
	}

	type testFile struct {
		env.File
		content interface{}
	}

	var (
		structure Outer

		cfgFile   testFile
		testFlags *pflag.FlagSet

		environment env.Environment
		err         error
	)

	generatedPFlagsSet := func(s interface{}) *pflag.FlagSet {
		set := pflag.NewFlagSet("testflags", pflag.ExitOnError)
		env.CreatePFlags(set, s)

		return set
	}

	standardPFlagsSet := func(s Outer) *pflag.FlagSet {
		set := pflag.NewFlagSet("testflags", pflag.ExitOnError)

		set.Bool(keyWbool, s.WBool, description)
		set.Int(keyWint, s.WInt, description)
		set.String(keyWstring, s.WString, description)
		set.String(keyWmappedVal, s.WMappedVal, description)

		set.Bool(keyNbool, s.Nest.NBool, description)
		set.Int(keyNint, s.Nest.NInt, description)
		set.String(keyNstring, s.Nest.NString, description)
		set.StringSlice(keyNslice, s.Nest.NSlice, description)
		set.String(keyNmappedVal, s.Nest.NMappedVal, description)

		return set
	}

	singlePFlagSet := func(key, defaultValue, description string) *pflag.FlagSet {
		set := pflag.NewFlagSet("testflags", pflag.ExitOnError)
		set.String(key, defaultValue, description)

		return set
	}

	setPFlags := func(o Outer) {
		Expect(testFlags.Set(keyWbool, cast.ToString(o.WBool))).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyWint, cast.ToString(o.WInt))).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyWstring, o.WString)).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyWmappedVal, o.WMappedVal)).ShouldNot(HaveOccurred())

		Expect(testFlags.Set(keyNbool, cast.ToString(o.Nest.NBool))).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyNint, cast.ToString(o.Nest.NInt))).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyNstring, o.Nest.NString)).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyNmappedVal, o.Nest.NMappedVal)).ShouldNot(HaveOccurred())
	}

	setEnvVars := func() {
		Expect(os.Setenv(strings.ToTitle(keyWbool), cast.ToString(structure.WBool))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyWint), cast.ToString(structure.WInt))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyWstring), structure.WString)).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyWmappedVal), structure.WMappedVal)).ShouldNot(HaveOccurred())

		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyNbool), ".", "_", 1), cast.ToString(structure.Nest.NBool))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyNint), ".", "_", 1), cast.ToString(structure.Nest.NInt))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyNstring), ".", "_", 1), structure.Nest.NString)).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyNslice), ".", "_", 1), strings.Join(structure.Nest.NSlice, ","))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyNmappedVal), ".", "_", 1), structure.Nest.NMappedVal)).ShouldNot(HaveOccurred())
	}

	cleanUpEnvVars := func() {
		Expect(os.Unsetenv(strings.ToTitle(keyWbool))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.ToTitle(keyWint))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.ToTitle(keyWstring))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.ToTitle(keyWmappedVal))).ShouldNot(HaveOccurred())

		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyNbool), ".", "_", 1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyNint), ".", "_", 1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyNstring), ".", "_", 1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyNslice), ".", "_", 1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyNmappedVal), ".", "_", 1))).ShouldNot(HaveOccurred())

		Expect(os.Unsetenv(strings.ToTitle(key))).ShouldNot(HaveOccurred())
	}

	cleanUpFlags := func() {
		pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
		testFlags = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	}

	createEnv := func() error {
		if cfgFile.content != nil {
			f := cfgFile.Location + string(filepath.Separator) + cfgFile.Name + "." + cfgFile.Format
			bytes, err := yaml.Marshal(cfgFile.content)
			Expect(err).ShouldNot(HaveOccurred())
			err = ioutil.WriteFile(f, bytes, 0640)
			Expect(err).ShouldNot(HaveOccurred())

			defer func() {
				os.Remove(f)
			}()
		}

		environment, err = env.New(testFlags)
		return err
	}

	verifyEnvCreated := func() {
		Expect(createEnv()).ShouldNot(HaveOccurred())
	}

	verifyEnvContainsValues := func(expected interface{}) {
		props := structs.Map(expected)
		for key, expectedValue := range props {
			switch v := expectedValue.(type) {
			case map[string]interface{}:
				for nestedKey, nestedExpectedValue := range v {
					expectedValue, ok := nestedExpectedValue.([]string)
					if ok {
						nestedExpectedValue = strings.Join(expectedValue, ",")
					}

					envValue := environment.Get(key + "." + nestedKey)
					switch actualValue := envValue.(type) {
					case []string:
						envValue = strings.Join(actualValue, ",")
					case []interface{}:
						temp := make([]string, len(actualValue))
						for i, v := range actualValue {
							temp[i] = fmt.Sprint(v)
						}
						envValue = strings.Join(temp, ",")
					}

					Expect(cast.ToString(envValue)).Should(Equal(cast.ToString(nestedExpectedValue)))
				}
			default:
				Expect(cast.ToString(environment.Get(key))).To(Equal(cast.ToString(expectedValue)))
			}
		}
	}

	BeforeEach(func() {
		testFlags = env.EmptyFlagSet()

		structure = Outer{
			WBool:      true,
			WInt:       1234,
			WString:    "wstringval",
			WMappedVal: "wmappedval",
			Nest: Nest{
				NBool:      true,
				NInt:       4321,
				NString:    "nstringval",
				NSlice:     []string{"nval1", "nval2", "nval3"},
				NMappedVal: "nmappedval",
			},
		}
	})

	AfterEach(func() {
		cleanUpEnvVars()
		cleanUpFlags()
	})

	Describe("New", func() {
		const (
			keyFileName     = "file.name"
			keyFileLocation = "file.location"
			keyFileFormat   = "file.format"
		)

		It("adds viper bindings for the provided flags", func() {
			testFlags.AddFlagSet(standardPFlagsSet(structure))

			verifyEnvCreated()

			verifyEnvContainsValues(structure)
		})

		Context("when SM config file exists", func() {
			BeforeEach(func() {
				cfgFile = testFile{
					File:    env.DefaultConfigFile(),
					content: structure,
				}
			})

			Context("when SM config file pflags are not provided", func() {
				BeforeEach(func() {
					Expect(testFlags.Lookup(keyFileName)).Should(BeNil())
					Expect(testFlags.Lookup(keyFileLocation)).Should(BeNil())
					Expect(testFlags.Lookup(keyFileFormat)).Should(BeNil())
				})

				It("returns no error", func() {
					verifyEnvCreated()

					Expect(environment.Get(keyFileName)).Should(BeNil())
					Expect(environment.Get(keyFileName)).Should(BeNil())
					Expect(environment.Get(keyFileName)).Should(BeNil())
				})

			})

			Context("when SM config file pflags are provided", func() {
				BeforeEach(func() {
					config.AddPFlags(testFlags)
				})

				It("allows obtaining SM config file values from the environment", func() {
					verifyEnvCreated()

					verifyEnvContainsValues(struct{ File env.File }{File: cfgFile.File})

				})

				It("allows unmarshaling SM config file values from the environment", func() {
					verifyEnvCreated()

					file := testFile{}
					Expect(environment.Unmarshal(&file)).ShouldNot(HaveOccurred())
					Expect(file.File).Should(Equal(cfgFile.File))
				})

				It("allows overriding the config file properties", func() {
					cfgFile.Name = "updatedName"
					testFlags.Set(keyFileName, "updatedName")
					verifyEnvCreated()

					verifyEnvContainsValues(struct{ File env.File }{File: cfgFile.File})
				})

				It("reads the file in the environment", func() {
					verifyEnvCreated()

					verifyEnvContainsValues(structure)
				})

				It("returns an err if config file loading fails", func() {
					cfgFile.Format = "json"
					testFlags.Set(keyFileFormat, "json")

					Expect(createEnv()).Should(HaveOccurred())
				})
			})
		})

		Context("when SM config file doesn't exist", func() {
			It("returns no error", func() {
				_, err := env.New(testFlags)
				Expect(err).ShouldNot(HaveOccurred())
			})
		})
	})

	Describe("BindPFlag", func() {
		const (
			key         = "test_flag"
			description = description
			aliasKey    = "test.flag"
		)
		It("allows getting a pflag from the environment with an alias name", func() {
			testFlags.AddFlagSet(singlePFlagSet(key, flagDefaultValue, description))

			verifyEnvCreated()

			Expect(environment.Get(key)).To(Equal(flagDefaultValue))
			Expect(environment.Get(aliasKey)).To(BeNil())

			environment.BindPFlag(aliasKey, testFlags.Lookup(key))

			Expect(environment.Get(key)).To(Equal(flagDefaultValue))
			Expect(environment.Get(aliasKey)).To(Equal(flagDefaultValue))
		})
	})

	Describe("Get", func() {
		var overrideStructure Outer

		BeforeEach(func() {
			overrideStructure = Outer{
				WBool:      false,
				WInt:       8888,
				WString:    "overrideval",
				WMappedVal: "overrideval",
				Nest: Nest{
					NBool:      false,
					NInt:       9999,
					NString:    "overrideval",
					NSlice:     []string{"nval1", "nval2", "nval3"},
					NMappedVal: "overrideval",
				},
			}
		})

		JustBeforeEach(func() {
			verifyEnvCreated()
		})

		Context("when properties are loaded via standard pflags", func() {
			BeforeEach(func() {
				testFlags.AddFlagSet(standardPFlagsSet(structure))
			})

			It("returns the default flag value if the flag is not set", func() {
				verifyEnvContainsValues(structure)
			})

			It("returns the flags values if the flags are set", func() {
				setPFlags(overrideStructure)

				verifyEnvContainsValues(overrideStructure)

			})
		})

		Context("when properties are loaded via generated pflags", func() {
			BeforeEach(func() {
				testFlags.AddFlagSet(generatedPFlagsSet(structure))
			})

			It("returns the default flag value if the flag is not set", func() {
				verifyEnvContainsValues(structure)
			})

			It("returns the flags values if the flags are set", func() {
				setPFlags(overrideStructure)

				verifyEnvContainsValues(overrideStructure)
			})
		})

		Context("when properties are loaded via SM config file", func() {
			BeforeEach(func() {
				cfgFile = testFile{
					File:    env.DefaultConfigFile(),
					content: structure,
				}
				config.AddPFlags(testFlags)
				verifyEnvCreated()
			})

			It("returns values from the config file", func() {
				verifyEnvContainsValues(structure)
			})
		})

		Context("when properties are loaded via OS environment variables", func() {
			BeforeEach(func() {
				setEnvVars()
			})

			It("returns values from environment", func() {
				verifyEnvContainsValues(structure)
			})
		})

		Context("override > pflag set > environment > file > pflag default", func() {
			BeforeEach(func() {
				testFlags.AddFlagSet(singlePFlagSet(key, flagDefaultValue, description))
			})

			It("respects loading order", func() {
				Expect(environment.Get(key)).Should(Equal(flagDefaultValue))

				config.AddPFlags(testFlags)
				cfgFile = testFile{
					File: env.DefaultConfigFile(),
					content: map[string]interface{}{
						key: fileValue,
					},
				}
				verifyEnvCreated()
				Expect(environment.Get(key)).Should(Equal(fileValue))

				os.Setenv(strings.ToTitle(key), envValue)
				Expect(environment.Get(key)).Should(Equal(envValue))

				testFlags.Set(key, flagValue)
				Expect(environment.Get(key)).Should(Equal(flagValue))

				environment.Set(key, overrideValue)
				Expect(environment.Get(key)).Should(Equal(overrideValue))
			})
		})
	})

	Describe("Set", func() {
		It("adds the property in the environment abstraction", func() {
			verifyEnvCreated()
			environment.Set(key, overrideValue)

			Expect(environment.Get(key)).To(Equal(overrideValue))
		})

		It("has highest priority", func() {
			testFlags.AddFlagSet(singlePFlagSet(key, flagDefaultValue, description))
			os.Setenv(key, envValue)
			verifyEnvCreated()
			testFlags.Set(key, flagValue)

			environment.Set(key, overrideValue)

			Expect(environment.Get(key)).Should(Equal(overrideValue))
		})
	})

	Describe("Unmarshal", func() {
		var actual Outer

		BeforeEach(func() {
			actual = Outer{}
		})

		JustBeforeEach(func() {
			verifyEnvCreated()
		})

		Context("when parameter is not a pointer to a struct", func() {
			It("returns an error", func() {
				Expect(environment.Unmarshal(actual)).To(Not(BeNil()))
			})

			It("returns an error", func() {
				Expect(environment.Unmarshal(10)).To(Not(BeNil()))
			})
		})

		Context("parameter is a pointer to a struct", func() {
			verifyUnmarshallingIsCorrect := func(actual, expected interface{}) {
				err := environment.Unmarshal(actual)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(actual).To(Equal(expected))
			}

			Context("when properties are loaded via standard pflags", func() {
				BeforeEach(func() {
					testFlags.AddFlagSet(standardPFlagsSet(structure))
				})

				It("unmarshals correctly", func() {
					verifyUnmarshallingIsCorrect(&actual, &structure)
				})
			})

			Context("when properties are loaded via generated pflags", func() {
				BeforeEach(func() {
					testFlags.AddFlagSet(generatedPFlagsSet(structure))
				})

				It("unmarshals correctly", func() {
					verifyUnmarshallingIsCorrect(&actual, &structure)
				})
			})

			Context("when property is loaded via config file", func() {
				BeforeEach(func() {
					cfgFile = testFile{
						File:    env.DefaultConfigFile(),
						content: structure,
					}
					config.AddPFlags(testFlags)
				})

				It("unmarshals correctly", func() {
					verifyUnmarshallingIsCorrect(&actual, &structure)
				})
			})

			Context("when properties are loaded via OS environment variables", func() {
				BeforeEach(func() {
					setEnvVars()
				})

				It("unmarshals correctly", func() {
					verifyUnmarshallingIsCorrect(&actual, &structure)
				})
			})

			Context("override > pflag set > environment > file > pflag default", func() {
				type s struct {
					Key string `mapstructure:"key"`
				}

				var str s

				BeforeEach(func() {
					str = s{}
					testFlags.AddFlagSet(singlePFlagSet(key, flagDefaultValue, ""))
				})

				It("respects loading order", func() {
					verifyUnmarshallingIsCorrect(&str, &s{flagDefaultValue})

					cfgFile = testFile{
						File: env.DefaultConfigFile(),
						content: map[string]interface{}{
							key: fileValue,
						},
					}
					config.AddPFlags(testFlags)
					verifyEnvCreated()
					verifyUnmarshallingIsCorrect(&str, &s{fileValue})

					os.Setenv(strings.ToTitle(key), envValue)
					verifyUnmarshallingIsCorrect(&str, &s{envValue})
					Expect(environment.Get(key)).Should(Equal(envValue))

					testFlags.Set(key, flagValue)
					verifyUnmarshallingIsCorrect(&str, &s{flagValue})

					environment.Set(key, overrideValue)
					verifyUnmarshallingIsCorrect(&str, &s{overrideValue})
				})
			})
		})
	})
})
