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

// Package env contains logic for working with env, flags and file configs via Viper
package config

import (
	"fmt"
	"strings"

	"github.com/fatih/structs"
	"github.com/spf13/cast"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Environment represents an abstraction over the env from which Service Manager configuration will be loaded
//go:generate counterfeiter . Environment
type Environment interface {
	Load() error
	Get(key string) interface{}
	Set(key string, value interface{})
	Unmarshal(value interface{}) error
	CreatePFlags(value interface{}) error
	BindPFlag(key string, flag *pflag.Flag) error
}

// File describes the name, path and the format of the file to be used to load the configuration in the env
type File struct {
	Name     string
	Location string
	Format   string
}

// DefaultFile holds the default SM config file properties
func DefaultFile() File {
	return File{
		Name:     "application",
		Location: ".",
		Format:   "yml",
	}
}

type cfg struct {
	File File
}

type viperEnv struct {
	Viper     *viper.Viper
	envPrefix string
}

// NewEnv returns a new application env loaded from the given configuration file with variables prefixed by the given prefix
func NewEnv(prefix string) Environment {
	return &viperEnv{
		Viper:     viper.New(),
		envPrefix: prefix,
	}
}

// Load prepares the environment for usage. It should be called after all relevant pflags have been created.
func (v *viperEnv) Load() error {
	v.Viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.Viper.SetEnvPrefix(v.envPrefix)
	v.Viper.AutomaticEnv()

	cfg := cfg{
		File: DefaultFile(),
	}

	// create and bind flags for providing SM config file using a structure - this creates the pflags using default
	// values from the structure and also binds them to viper.
	v.CreatePFlags(cfg)

	// bind any flags that were added using the standard way and not via the CreatePFlags(interface{}) method.
	// This way, all pflags that are also available for retrieval via env Get using their names
	pflag.CommandLine.VisitAll(func(flag *pflag.Flag) {
		v.Viper.BindPFlag(flag.Name, flag)
	})

	pflag.Parse()

	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("could not find configuration cfg: %s", err)
	}

	v.Viper.AddConfigPath(cfg.File.Location)
	v.Viper.SetConfigName(cfg.File.Name)
	v.Viper.SetConfigType(cfg.File.Format)
	if err := v.Viper.ReadInConfig(); err != nil {
		return fmt.Errorf("could not read configuration cfg: %s", err)
	}

	// allows nested properties loaded from config files to be accessible with underscore separator as well as with dot
	for _, key := range v.Viper.AllKeys() {
		alias := strings.Replace(key, ".", "_", -1)
		if key != alias {
			v.Viper.RegisterAlias(alias, key)
		}
	}
	return nil
}

// Get exposes viper's Get
func (v *viperEnv) Get(key string) interface{} {
	return v.Viper.Get(key)
}

// Set exposes viper's Set
func (v *viperEnv) Set(key string, value interface{}) {
	v.Viper.Set(key, value)
}

// Unmarshal exposes viper's Unmarshal. Prior to unmarshaling it creates the necessary pflag and env var bindings
// so that pflag / env var values are also used during the unmarshaling.
func (v *viperEnv) Unmarshal(value interface{}) error {
	if err := v.prepareBindings(value); err != nil {
		return err
	}
	return v.Viper.Unmarshal(value)
}

// prepareBindings binds the structure's fields as env variables and pflags to viper
// so that viper knows to look for them during unmarshaling.
func (v *viperEnv) prepareBindings(value interface{}) error {
	properties := make(map[string]interface{})

	traverseFields(value, "", properties)
	for bindingFlagName := range properties {
		// let viper's AllKeys know about the env variables that would be used during unmarshaling
		if err := v.Viper.BindEnv(bindingFlagName); err != nil {
			return err
		}

		// PFlags are already bound with underscore separator on nest points - however during unmarshaling viper
		// is using dot as separator for nested paths in structures - therefore additional pflag bindings are required.
		key := strings.Replace(bindingFlagName, ".", "_", -1)
		flag := pflag.Lookup(key)
		if flag != nil {
			if err := v.Viper.BindPFlag(bindingFlagName, flag); err != nil {
				return err
			}
		}
	}
	return nil
}

// CreatePFlags creates pflags for the structure's fields. If values are present within the structure, they will be used
// as pflag default values. PFlags can also be manually added using the standard way (not using this method).
// If a pflag with the same name was already added using the standard way, this method will not override it.
// Creating pflags should be done before calling Load() - pflags created after loading will be ignored.
func (v *viperEnv) CreatePFlags(value interface{}) error {
	properties := make(map[string]interface{})
	traverseFields(value, "", properties)
	for bindingName, defaultValue := range properties {
		//  underscores instead of dots in flag names
		flagName := strings.Replace(bindingName, ".", "_", -1)
		if pflag.Lookup(flagName) == nil {
			pflag.String(flagName, cast.ToString(defaultValue), fmt.Sprintf("commandline argument for %s", flagName))
		}
	}
	return nil
}

// BindPFlag allows binding a single flag to the env
func (v *viperEnv) BindPFlag(key string, value *pflag.Flag) error {
	return v.Viper.BindPFlag(key, value)
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
		if field.IsExported() {
			var name string
			if field.Tag("mapstructure") != "" {
				name = field.Tag("mapstructure")
			} else {
				name = field.Name()
			}
			if !field.IsEmbedded() {
				buffer += name + "."
			}
			traverseFields(field.Value(), buffer, result)
			if !field.IsEmbedded() {
				buffer = buffer[0:strings.LastIndex(buffer, name)]
			}
		}
	}
}
