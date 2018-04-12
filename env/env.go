/*
 *    Copyright 2018 The Service Manager Authors
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
	"strings"

	"github.com/fatih/structs"
	"github.com/spf13/viper"
	"github.com/Peripli/service-manager/server"
)

type viperEnv struct {
	Viper      *viper.Viper
	configFile *configFile
	envPrefix  string
}

type configFile struct {
	Name   string
	Path   string
	Format string
}

func Default() server.Environment {
	envPrefix := "SM"
	configFile := &configFile{
		Path:   ".",
		Name:   "application",
		Format: "yml",
	}
	return New(configFile, envPrefix)
}

func New(file *configFile, envPrefix string) server.Environment {
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
		return fmt.Errorf("Could not read configuration file: %s", err)
	}
	return nil
}

func (v *viperEnv) Get(key string) interface{} {
	return v.Viper.Get(key)
}

func (v *viperEnv) Unmarshal(value interface{}) error {
	if err := v.introduce(value); err != nil {
		return err
	}
	return v.Viper.Unmarshal(value)
}

// introduce introduces the structure's fields as viper properties
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
