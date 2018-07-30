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

package env

import (
	"fmt"
	"reflect"
	"strings"

	"os"

	"flag"

	"github.com/fatih/structs"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// File describes the name, path and the format of the file to be used to load the configuration in the env
type File struct {
	Name     string
	Location string
	Format   string
}

// DefaultConfigFile holds the default SM config file properties
func DefaultConfigFile() File {
	return File{
		Name:     "application",
		Location: ".",
		Format:   "yml",
	}
}

// CreatePFlagsForConfigFile creates pflags for setting the configuration file
func CreatePFlagsForConfigFile(set *pflag.FlagSet) {
	CreatePFlags(set, struct{ File File }{File: DefaultConfigFile()})
}

// Environment represents an abstraction over the env from which Service Manager configuration will be loaded
//go:generate counterfeiter . Environment
type Environment interface {
	Get(key string) interface{}
	Set(key string, value interface{})
	Unmarshal(value interface{}) error
	BindPFlag(key string, flag *pflag.Flag) error
}

// ViperEnv represents an implementation of the Environment interface that uses viper
type ViperEnv struct {
	*viper.Viper
}

// EmptyFlagSet creates an empty flag set and adds the default se of flags to it
func EmptyFlagSet() *pflag.FlagSet {
	set := pflag.NewFlagSet("Service Manager Configuration Flags", pflag.ExitOnError)
	set.AddFlagSet(pflag.CommandLine)
	set.AddGoFlagSet(flag.CommandLine)
	return set
}

//CreatePFlags Creates pflags for the value structure and adds them in the provided set
func CreatePFlags(set *pflag.FlagSet, value interface{}) {
	properties := make(map[string]interface{})
	traverseFields(value, "", properties)
	for bindingName, defaultValue := range properties {
		if set.Lookup(bindingName) == nil {
			set.String(bindingName, cast.ToString(defaultValue), fmt.Sprintf("commandline argument for %s", bindingName))
		}
	}
}

// New creates a new environment. It accepts a flag set that should contain all the flags that the
// environment should be aware of.
func New(set *pflag.FlagSet) (*ViperEnv, error) {
	v := &ViperEnv{
		Viper: viper.New(),
	}
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := set.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	set.VisitAll(func(flag *pflag.Flag) {
		if err := v.BindPFlag(flag.Name, flag); err != nil {
			logrus.Panic(err)
		}
	})

	if err := v.setupConfigFile(); err != nil {
		return nil, err
	}

	return v, nil
}

// Unmarshal exposes viper's Unmarshal. Prior to unmarshaling it creates the necessary pflag and env var bindings
// so that pflag / env var values are also used during the unmarshaling.
func (v *ViperEnv) Unmarshal(value interface{}) error {
	properties := make(map[string]interface{})
	traverseFields(value, "", properties)
	for flagName := range properties {
		if err := v.Viper.BindEnv(flagName); err != nil {
			return err
		}
	}
	return v.Viper.Unmarshal(value)
}

// traverseFields traverses the provided structure and prepares a slice of strings that contains
// the paths to the structure fields (nested paths in the provided structure use dot as a separator)
func traverseFields(value interface{}, buffer string, result map[string]interface{}) {
	if !structs.IsStruct(value) {
		index := strings.LastIndex(buffer, ".")
		if index == -1 {
			index = 0
		}
		key := strings.ToLower(buffer[0:index])
		result[key] = value
		return
	}

	s := structs.New(value)
	for _, field := range s.Fields() {
		if field.IsExported() && field.Kind() != reflect.Interface && field.Kind() != reflect.Func {
			var name string
			if field.Tag("mapstructure") != "" {
				name = field.Tag("mapstructure")
			} else {
				name = field.Name()
			}
			buffer += name + "."
			traverseFields(field.Value(), buffer, result)
			buffer = buffer[0:strings.LastIndex(buffer, name)]
		}
	}
}

func (v *ViperEnv) setupConfigFile() error {
	cfg := struct{ File File }{File: File{}}
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("could not find configuration cfg: %s", err)
	}

	v.Viper.AddConfigPath(cfg.File.Location)
	v.Viper.SetConfigName(cfg.File.Name)
	v.Viper.SetConfigType(cfg.File.Format)

	if err := v.Viper.ReadInConfig(); err != nil {
		if err, ok := err.(viper.ConfigFileNotFoundError); ok {
			logrus.Info("Config File was not found: ", err)
			return nil
		}
		return fmt.Errorf("could not read configuration cfg: %s", err)
	}
	return nil
}
