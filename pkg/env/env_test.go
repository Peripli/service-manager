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
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/fatih/structs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"testing"

	"path/filepath"

	"strings"

	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env"
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
		mapKey           = "mapkey"
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

		keySquashNbool      = "nbool"
		keySquashNint       = "nint"
		keySquashNstring    = "nstring"
		keySquashNslice     = "nslice"
		keySquashNmappedVal = "n_mapped_val"

		keyMapNbool      = "wmapnest" + "." + mapKey + "." + "nbool"
		keyMapNint       = "wmapnest" + "." + mapKey + "." + "nint"
		keyMapNstring    = "wmapnest" + "." + mapKey + "." + "nstring"
		keyMapNslice     = "wmapnest" + "." + mapKey + "." + "nslice"
		keyMapNmappedVal = "wmapnest" + "." + mapKey + "." + "n_mapped_val"

		keyLogFormat = "log.format"
		keyLogLevel  = "log.level"
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
		WMapNest   map[string]Nest
		Nest       Nest
		Squash     Nest `mapstructure:",squash"`
		Log        log.Settings
	}

	type FlatOuter struct {
		WBool      bool
		WInt       int
		WString    string
		WMappedVal string `mapstructure:"w_mapped_val" structs:"w_mapped_val" yaml:"w_mapped_val"`
		WMapNest   map[string]Nest
		Nest       Nest

		// Flattened Nest fields due to squash tag
		NBool      bool
		NInt       int
		NString    string
		NSlice     []string
		NMappedVal string `mapstructure:"n_mapped_val" structs:"n_mapped_val"  yaml:"n_mapped_val"`

		Log log.Settings
	}

	type testFile struct {
		env.File
		content interface{}
	}

	var (
		outer     Outer
		flatOuter FlatOuter

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

		set.Bool(keySquashNbool, s.Squash.NBool, description)
		set.Int(keySquashNint, s.Squash.NInt, description)
		set.String(keySquashNstring, s.Squash.NString, description)
		set.StringSlice(keySquashNslice, s.Squash.NSlice, description)
		set.String(keySquashNmappedVal, s.Squash.NMappedVal, description)

		set.Bool(keyNbool, s.Nest.NBool, description)
		set.Int(keyNint, s.Nest.NInt, description)
		set.String(keyNstring, s.Nest.NString, description)
		set.StringSlice(keyNslice, s.Nest.NSlice, description)
		set.String(keyNmappedVal, s.Nest.NMappedVal, description)

		set.Bool(keyMapNbool, s.WMapNest[mapKey].NBool, description)
		set.Int(keyMapNint, s.WMapNest[mapKey].NInt, description)
		set.String(keyMapNstring, s.WMapNest[mapKey].NString, description)
		set.StringSlice(keyMapNslice, s.WMapNest[mapKey].NSlice, description)
		set.String(keyMapNmappedVal, s.WMapNest[mapKey].NMappedVal, description)

		set.String(keyLogLevel, s.Log.Level, description)
		set.String(keyLogFormat, s.Log.Format, description)

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

		Expect(testFlags.Set(keySquashNbool, cast.ToString(o.Squash.NBool))).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keySquashNint, cast.ToString(o.Squash.NInt))).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keySquashNstring, o.Squash.NString)).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keySquashNslice, strings.Join(o.Squash.NSlice, ","))).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keySquashNmappedVal, o.Squash.NMappedVal)).ShouldNot(HaveOccurred())

		Expect(testFlags.Set(keyNbool, cast.ToString(o.Nest.NBool))).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyNint, cast.ToString(o.Nest.NInt))).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyNstring, o.Nest.NString)).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyNmappedVal, o.Nest.NMappedVal)).ShouldNot(HaveOccurred())

		Expect(testFlags.Set(keyMapNbool, cast.ToString(o.WMapNest[mapKey].NBool))).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyMapNint, cast.ToString(o.WMapNest[mapKey].NInt))).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyMapNstring, o.WMapNest[mapKey].NString)).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyMapNmappedVal, o.WMapNest[mapKey].NMappedVal)).ShouldNot(HaveOccurred())

		Expect(testFlags.Set(keyLogFormat, o.Log.Format)).ShouldNot(HaveOccurred())
		Expect(testFlags.Set(keyLogLevel, o.Log.Level)).ShouldNot(HaveOccurred())
	}

	setEnvVars := func() {
		Expect(os.Setenv(strings.ToTitle(keyWbool), cast.ToString(outer.WBool))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyWint), cast.ToString(outer.WInt))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyWstring), outer.WString)).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keyWmappedVal), outer.WMappedVal)).ShouldNot(HaveOccurred())

		Expect(os.Setenv(strings.ToTitle(keySquashNbool), cast.ToString(outer.Squash.NBool))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keySquashNint), cast.ToString(outer.Squash.NInt))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keySquashNstring), outer.Squash.NString)).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keySquashNslice), strings.Join(outer.Squash.NSlice, ","))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.ToTitle(keySquashNmappedVal), outer.Squash.NMappedVal)).ShouldNot(HaveOccurred())

		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyNbool), ".", "_", -1), cast.ToString(outer.Nest.NBool))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyNint), ".", "_", -1), cast.ToString(outer.Nest.NInt))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyNstring), ".", "_", -1), outer.Nest.NString)).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyNslice), ".", "_", -1), strings.Join(outer.Nest.NSlice, ","))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyNmappedVal), ".", "_", -1), outer.Nest.NMappedVal)).ShouldNot(HaveOccurred())

		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyMapNbool), ".", "_", -1), cast.ToString(outer.WMapNest[mapKey].NBool))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyMapNint), ".", "_", -1), cast.ToString(outer.WMapNest[mapKey].NInt))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyMapNstring), ".", "_", -1), outer.WMapNest[mapKey].NString)).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyMapNslice), ".", "_", -1), strings.Join(outer.WMapNest[mapKey].NSlice, ","))).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyMapNmappedVal), ".", "_", -1), outer.WMapNest[mapKey].NMappedVal)).ShouldNot(HaveOccurred())

		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyLogFormat), ".", "_", -1), outer.Log.Format)).ShouldNot(HaveOccurred())
		Expect(os.Setenv(strings.Replace(strings.ToTitle(keyLogLevel), ".", "_", -1), outer.Log.Level)).ShouldNot(HaveOccurred())
	}

	cleanUpEnvVars := func() {
		Expect(os.Unsetenv(strings.ToTitle(keyWbool))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.ToTitle(keyWint))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.ToTitle(keyWstring))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.ToTitle(keyWmappedVal))).ShouldNot(HaveOccurred())

		Expect(os.Unsetenv(strings.ToTitle(keySquashNbool))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.ToTitle(keySquashNint))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.ToTitle(keySquashNstring))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.ToTitle(keySquashNslice))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.ToTitle(keySquashNmappedVal))).ShouldNot(HaveOccurred())

		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyNbool), ".", "_", -1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyNint), ".", "_", -1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyNstring), ".", "_", -1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyNslice), ".", "_", -1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyNmappedVal), ".", "_", -1))).ShouldNot(HaveOccurred())

		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyMapNbool), ".", "_", -1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyMapNint), ".", "_", -1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyMapNstring), ".", "_", -1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyMapNslice), ".", "_", -1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyMapNmappedVal), ".", "_", -1))).ShouldNot(HaveOccurred())

		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyLogFormat), ".", "_", -1))).ShouldNot(HaveOccurred())
		Expect(os.Unsetenv(strings.Replace(strings.ToTitle(keyLogLevel), ".", "_", -1))).ShouldNot(HaveOccurred())

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
		}

		environment, err = env.DefaultLogger(context.TODO(), func(set *pflag.FlagSet) {
			set.AddFlagSet(testFlags)
		})
		return err
	}

	cleanUpFile := func() {
		f := cfgFile.Location + string(filepath.Separator) + cfgFile.Name + "." + cfgFile.Format
		os.Remove(f)
	}

	verifyEnvCreated := func() {
		Expect(createEnv()).ShouldNot(HaveOccurred())
	}

	var verifyValues func(expected map[string]interface{}, prefix string)

	verifyValues = func(fields map[string]interface{}, prefix string) {
		for name, value := range fields {
			switch v := value.(type) {
			case map[string]interface{}:
				verifyValues(v, prefix+name+".")
			case []string:
				switch envVar := environment.Get(prefix + name).(type) {
				case string:
					Expect(envVar).To(Equal(strings.Join(v, ",")))
				case []string, []interface{}:
					Expect(fmt.Sprint(envVar)).To(Equal(fmt.Sprint(v)))
				default:
					Fail(fmt.Sprintf("Expected env value of type []string but got: %T", envVar))
				}
			default:
				Expect(cast.ToString(environment.Get(prefix+name))).To(Equal(cast.ToString(v)), prefix+name)
			}
		}
	}

	verifyEnvContainsValues := func(expected interface{}) {
		fields := structs.Map(expected)
		verifyValues(fields, "")
	}

	BeforeEach(func() {
		testFlags = env.EmptyFlagSet()

		nest := Nest{
			NBool:      true,
			NInt:       4321,
			NString:    "nstringval",
			NSlice:     []string{"nval1", "nval2", "nval3"},
			NMappedVal: "nmappedval",
		}

		outer = Outer{
			WBool:      true,
			WInt:       1234,
			WString:    "wstringval",
			WMappedVal: "wmappedval",
			Squash:     nest,
			Log: log.Settings{
				Level:  "error",
				Format: "text",
			},
			Nest: nest,
			WMapNest: map[string]Nest{
				mapKey: nest,
			},
		}

		flatOuter = FlatOuter{
			WBool:      true,
			WInt:       1234,
			WString:    "wstringval",
			WMappedVal: "wmappedval",
			NBool:      true,
			NInt:       4321,
			NString:    "nstringval",
			NSlice:     []string{"nval1", "nval2", "nval3"},
			NMappedVal: "nmappedval",
			Log: log.Settings{
				Level:  "error",
				Format: "text",
			},
			Nest: nest,
			WMapNest: map[string]Nest{
				mapKey: nest,
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
			testFlags.AddFlagSet(standardPFlagsSet(outer))
			cfgFile.content = nil

			verifyEnvCreated()

			verifyEnvContainsValues(flatOuter)
		})

		Context("when SM config file exists", func() {
			BeforeEach(func() {
				cfgFile = testFile{
					File:    env.DefaultConfigFile(),
					content: flatOuter,
				}
			})

			AfterEach(func() {
				cleanUpFile()
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

					verifyEnvContainsValues(flatOuter)
				})

				It("returns an err if config file loading fails", func() {
					cfgFile.Format = "json"
					testFlags.Set(keyFileFormat, "json")

					Expect(createEnv()).Should(HaveOccurred())
				})

				Context("when the logging properties are changed", func() {
					It("reconfigures the loggers with the correct logging config", func() {
						verifyEnvCreated()
						oldCfg := log.Configuration()
						newLogLevel := logrus.DebugLevel.String()
						Expect(newLogLevel).ToNot(Equal(oldCfg.Level))
						Expect(log.D().Logger.Level.String()).ToNot(Equal(newLogLevel))
						newOutput := os.Stderr.Name()
						Expect(newOutput).ToNot(Equal(oldCfg.Output))
						Expect(log.D().Logger.Out.(*os.File).Name()).ToNot(Equal(newOutput))

						f := cfgFile.Location + string(filepath.Separator) + cfgFile.Name + "." + cfgFile.Format
						fileContent := cfgFile.content.(FlatOuter)
						fileContent.Log.Level = logrus.DebugLevel.String()
						fileContent.Log.Output = newOutput
						cfgFile.content = fileContent
						bytes, err := yaml.Marshal(cfgFile.content)
						Expect(err).ShouldNot(HaveOccurred())
						err = ioutil.WriteFile(f, bytes, 0640)
						Expect(err).ShouldNot(HaveOccurred())

						Eventually(func() bool {
							return log.D().Logger.IsLevelEnabled(logrus.DebugLevel)
						}).Should(BeTrue())
						Expect(log.Configuration().Level).To(Equal(newLogLevel))
						Expect(log.Configuration().Output).ToNot(Equal(newOutput))
					})
				})
			})
		})

		Context("when SM config file doesn't exist", func() {
			It("returns no error", func() {
				_, err := env.New(context.TODO(), testFlags)
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

		AfterEach(func() {
			cleanUpFile()
		})

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
		var overrideStructureOutput FlatOuter

		BeforeEach(func() {
			nest := Nest{
				NBool:      false,
				NInt:       9999,
				NString:    "overrideval",
				NSlice:     []string{"nval1", "nval2", "nval3"},
				NMappedVal: "overrideval",
			}

			overrideStructure = Outer{
				WBool:      false,
				WInt:       8888,
				WString:    "overrideval",
				WMappedVal: "overrideval",
				Nest:       nest,
				Squash:     nest,
			}

			overrideStructureOutput = FlatOuter{
				WBool:      false,
				WInt:       8888,
				WString:    "overrideval",
				WMappedVal: "overrideval",
				Nest:       nest,
				NBool:      false,
				NInt:       9999,
				NString:    "overrideval",
				NSlice:     []string{"nval1", "nval2", "nval3"},
				NMappedVal: "overrideval",
			}
		})

		AfterEach(func() {
			cleanUpFile()
		})

		JustBeforeEach(func() {
			verifyEnvCreated()
		})

		Context("when properties are loaded via standard pflags", func() {
			BeforeEach(func() {
				testFlags.AddFlagSet(standardPFlagsSet(outer))
			})

			It("returns the default flag value if the flag is not set", func() {
				verifyEnvContainsValues(flatOuter)
			})

			It("returns the flags values if the flags are set", func() {
				setPFlags(overrideStructure)

				verifyEnvContainsValues(overrideStructureOutput)

			})
		})

		Context("when properties are loaded via generated pflags", func() {
			BeforeEach(func() {
				testFlags.AddFlagSet(generatedPFlagsSet(outer))
			})

			It("returns the default flag value if the flag is not set", func() {
				verifyEnvContainsValues(flatOuter)
			})

			It("returns the flags values if the flags are set", func() {
				setPFlags(overrideStructure)

				verifyEnvContainsValues(overrideStructureOutput)
			})
		})

		Context("when properties are loaded via SM config file", func() {
			BeforeEach(func() {
				cfgFile = testFile{
					File:    env.DefaultConfigFile(),
					content: flatOuter,
				}
				config.AddPFlags(testFlags)
				verifyEnvCreated()
			})

			It("returns values from the config file", func() {
				verifyEnvContainsValues(flatOuter)
			})
		})

		Context("when properties are loaded via OS environment variables", func() {
			BeforeEach(func() {
				setEnvVars()
			})

			It("returns values from environment", func() {
				verifyEnvContainsValues(flatOuter)
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
		AfterEach(func() {
			cleanUpFile()
		})

		It("adds the property in the environment abstraction", func() {
			verifyEnvCreated()
			environment.Set(key, overrideValue)

			Expect(environment.Get(key)).To(Equal(overrideValue))
		})

		It("has highest priority", func() {
			testFlags.AddFlagSet(singlePFlagSet(key, flagDefaultValue, description))
			Expect(os.Setenv(key, envValue)).ToNot(HaveOccurred())
			verifyEnvCreated()
			Expect(testFlags.Set(key, flagValue)).ToNot(HaveOccurred())

			environment.Set(key, overrideValue)

			Expect(environment.Get(key)).Should(Equal(overrideValue))
		})
	})

	Describe("Unmarshal", func() {
		var actual Outer

		BeforeEach(func() {
			actual = Outer{
				WMapNest: map[string]Nest{
					mapKey: {},
				},
			}
		})

		JustBeforeEach(func() {
			verifyEnvCreated()
		})

		AfterEach(func() {
			cleanUpFile()
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
					testFlags.AddFlagSet(standardPFlagsSet(outer))
				})

				It("unmarshals correctly", func() {
					verifyUnmarshallingIsCorrect(&actual, &outer)
				})
			})

			Context("when properties are loaded via generated pflags", func() {
				BeforeEach(func() {
					testFlags.AddFlagSet(generatedPFlagsSet(outer))
				})

				It("unmarshals correctly", func() {
					verifyUnmarshallingIsCorrect(&actual, &outer)
				})
			})

			Context("when property is loaded via config file", func() {
				BeforeEach(func() {
					cfgFile = testFile{
						File:    env.DefaultConfigFile(),
						content: flatOuter,
					}
					config.AddPFlags(testFlags)
				})

				It("unmarshals correctly", func() {
					verifyUnmarshallingIsCorrect(&actual, &outer)
				})
			})

			Context("when properties are loaded via OS environment variables", func() {
				BeforeEach(func() {
					setEnvVars()
				})

				It("unmarshals correctly", func() {
					verifyUnmarshallingIsCorrect(&actual, &outer)
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
