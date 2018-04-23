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
package env

import (
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/server"
	"github.com/fatih/structs"
	"github.com/spf13/viper"
)

type viperEnv struct {
	Viper      *viper.Viper
	configFile *ConfigFile
	envPrefix  string
}

// ConfigFile describes the name and the format of the file to be used to load the configuration in the environment
type ConfigFile struct {
	Name   string
	Path   string
	Format string
}

// Default returns the default environment configuration to be loaded from application.yml
func Default() server.Environment {
	envPrefix := "SM"
	configFile := &ConfigFile{
		Path:   ".",
		Name:   "application",
		Format: "yml",
	}
	return New(configFile, envPrefix)
}

// New returns a new application environment loaded from the given configuration file with variables prefixed by the given prefix
func New(file *ConfigFile, envPrefix string) server.Environment {
	return &viperEnv{
		Viper:      viper.New(),
		configFile: file,
		envPrefix:  envPrefix,
	}
}

func (v *viperEnv) Load() error {
	v.Viper.AddConfigPath(v.configFile.Path)
	v.Viper.SetConfigName(v.configFile.Name)
	v.Viper.SetConfigType(v.configFile.Format)
	v.Viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.Viper.SetEnvPrefix(v.envPrefix)
	v.Viper.AutomaticEnv()
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
	if err := v.introduce(value); err != nil {
		return err
	}
	return v.Viper.Unmarshal(value)
}

// introduce introduces the structure's fields as viper properties.
func (v *viperEnv) introduce(value interface{}) error {
	var properties []string
	traverseFields(value, "", &properties)
	for _, property := range properties {
		if err := viper.BindEnv(property); err != nil {
			return err
		}
	}
	return nil
}

// traverseFields traverses the provided structure and prepares a slice of strings that contains
//
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
