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
package server

import (
	"strings"

	"fmt"

	"github.com/fatih/structs"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	IntroducePFlags(struct{ File }{File: File{}})
}

// Environment represents an abstraction over the environment from which Service Manager configuration will be loaded
//go:generate counterfeiter . Environment
type Environment interface {
	Load() error
	Get(key string) interface{}
	Set(key string, value interface{})
	Unmarshal(value interface{}) error
}

// File describes the name, path and the format of the file to be used to load the configuration in the environment
type File struct {
	Name     string
	Location string
	Format   string
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

//TODO maybe pass to load a ... of interface{}, make them structs and introduceEnv them but s need to be already with fields and vals
func (v *viperEnv) Load() error {
	v.Viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.Viper.SetEnvPrefix(v.envPrefix)
	v.Viper.AutomaticEnv()

	pflag.Parse()
	pflag.CommandLine.VisitAll(func(flag *pflag.Flag) {
		bindingName := strings.Replace(flag.Name, "_", ".", -1)
		v.Viper.BindPFlag(bindingName, flag)
	})

	file := struct{ File }{File: File{}}
	if err := v.Unmarshal(&file); err != nil {
		return fmt.Errorf("could not find configuration file: %s", err)
	}

	v.Viper.AddConfigPath(file.Location)
	v.Viper.SetConfigName(file.Name)
	v.Viper.SetConfigType(file.Format)
	if err := v.Viper.ReadInConfig(); err != nil {
		return fmt.Errorf("could not read configuration file: %s", err)
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
	if err := v.introduceEnv(value); err != nil {
		return err
	}
	return v.Viper.Unmarshal(value)
}

// introduceEnv binds the structure's fields as environment variables to viper so that viper knows to look for them.
func (v *viperEnv) introduceEnv(value interface{}) error {
	var properties []string
	traverseFields(value, "", &properties)
	for _, property := range properties {
		if err := viper.BindEnv(property); err != nil {
			return err
		}
	}
	return nil
}

func IntroducePFlags(value interface{}) {
	var properties []string
	traverseFields(value, "", &properties)
	for _, bindingName := range properties {
		flagName := strings.Replace(bindingName, ".", "_", -1)
		if pflag.Lookup(flagName) == nil {
			pflag.String(flagName, "", "")
		}
	}
}

// traverseFields traverses the provided structure and prepares a slice of strings that contains
func traverseFields(value interface{}, buffer string, result *[]string) {
	if !structs.IsStruct(value) {
		index := strings.LastIndex(buffer, ".")
		if index == -1 {
			index = 0
		}
		*result = append(*result, strings.ToLower(buffer[0:index]))
		return
	}
	s := structs.New(value)
	for _, field := range s.Fields() {
		if field.IsExported() {
			if !field.IsEmbedded() {
				buffer += field.Name() + "."
			}
			traverseFields(field.Value(), buffer, result)
			if !field.IsEmbedded() {
				buffer = buffer[0:strings.LastIndex(buffer, field.Name())]
			}
		}
	}
}
