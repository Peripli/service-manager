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

// Package env contains logic for working with environment, flags and file configs via Viper
package config

import (
	"fmt"
	"strings"

	"github.com/fatih/structs"
	"github.com/spf13/cast"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	//pflag.String("file_name", "application", "The name of the config file")
	//pflag.String("file_location", ".", "The location of the config file")
	//pflag.String("file_format", "yml", "The format of the config file")
}

// Environment represents an abstraction over the environment from which Service Manager configuration will be loaded
//go:generate counterfeiter . Environment
type Environment interface {
	Load() error
	Get(key string) interface{}
	Set(key string, value interface{})
	Unmarshal(value interface{}) error
	BindPFlags(value interface{}) error
}

// File describes the name, path and the format of the file to be used to load the configuration in the environment
type File struct {
	Name     string
	Location string
	Format   string
}

type cfg struct {
	File File
}

type viperEnv struct {
	Viper     *viper.Viper
	envPrefix string
}

// NewEnv returns a new application environment loaded from the given configuration file with variables prefixed by the given prefix
func NewEnv(prefix string) Environment {
	return &viperEnv{
		Viper:     viper.New(),
		envPrefix: prefix,
	}
}

func (v *viperEnv) Load() error {
	v.Viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.Viper.SetEnvPrefix(v.envPrefix)
	v.Viper.AutomaticEnv()

	cfg := cfg{
		File: File{
			Name:     "application",
			Location: ".",
			Format:   "yml",
		},
	}

	v.BindPFlags(cfg)
	//pflag.CommandLine.VisitAll(func(flag *pflag.Flag) {
	//	bindingName := strings.Replace(flag.Name, "_", ".", -1)
	//	v.Viper.BindPFlag(bindingName, flag)
	//})
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

	return nil
}

func (v *viperEnv) Get(key string) interface{} {
	return v.Viper.Get(key)
}

func (v *viperEnv) Set(key string, value interface{}) {
	v.Viper.Set(key, value)
}

func (v *viperEnv) Unmarshal(value interface{}) error {
	if err := v.BindEnv(value); err != nil {
		return err
	}
	return v.Viper.Unmarshal(value)
}

// BindEnv binds the structure's fields as environment variables to viper so that viper knows to look for them.
func (v *viperEnv) BindEnv(value interface{}) error {
	properties := make(map[string]interface{})

	traverseFields(value, "", properties)
	for key := range properties {
		if err := viper.BindEnv(key); err != nil {
			return err
		}
	}
	return nil
}

// BindPFlags  binds pflags to viper for the fields of a given structure so that viper knows to look for them.
// Flags can be manually added using the standard way. If pflag with the same name was already added using the standard way,
// this method does not override them. Binding pflags should be done before loading the environment, otherwise pflags
// bound after that will be ignored.
func (v *viperEnv) BindPFlags(value interface{}) error {
	properties := make(map[string]interface{})

	traverseFields(value, "", properties)
	for bindingName, defaultValue := range properties {
		flagName := strings.Replace(bindingName, ".", "_", -1)
		if pflag.Lookup(flagName) == nil {
			pflag.String(flagName, cast.ToString(defaultValue), fmt.Sprintf("commandline argument for %s", flagName))
		}
		if err := v.Viper.BindPFlag(bindingName, pflag.Lookup(flagName)); err != nil {
			return err
		}
	}
	return nil
}

// traverseFields traverses the provided structure and prepares a slice of strings that contains
// the paths to the structure fields
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
