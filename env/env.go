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
)

type viperEnv struct {
	Viper      *viper.Viper
	configFile *ConfigFile
	envPrefix  string
}

type ConfigFile struct {
	Name   string
	Path   string
	Format string
}

//var _ server.environment = &viperEnv{}

func Default() *viperEnv {
	envPrefix := "SM"
	configFile := &ConfigFile{
		Path:   ".",
		Name:   "application",
		Format: "yml",
	}
	return New(configFile, envPrefix)
}

func New(file *ConfigFile, envPrefix string) *viperEnv {
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
		panic(fmt.Sprintf("Could not read configuration file: %s", err))
	}
	return nil
}

func (v *viperEnv) Get(key string) interface{} {
	return v.Viper.Get(key)
}

func (v *viperEnv) Unmarshal(value interface{}) error {
	if err := bindStruct(v.Viper, value); err != nil {
		return err
	}
	return v.Viper.Unmarshal(value)
}

//THIS IS OPTIONAL but automates something that is missing from viper -
// If we pass a struct to be unmarshaled and we want some of the fields to be loaded
// from env variables we would need to perform something manual to let viper know about each of these fields.
// We would either need to do one of these:
// specify default values for the fields in application.yml, pass default values as flags,
// call viper.SetDefault(x.y) where x.y in env should be [PREFIX_]X_Y, call viper.BindEnv(x.y)
//TODO method below needs refactoring
func bindStruct(viper *viper.Viper, value interface{}) error {
	var result = []string{}
	var a func(value interface{}, buffer string)
	a = func(value interface{}, buffer string) {
		if !structs.IsStruct(value) {
			index := strings.LastIndex(buffer, ".")
			if index == -1 {
				index = 0
			}
			result = append(result, strings.ToLower(buffer[0:index]))
			return
		}
		s := structs.New(value)
		for _, field := range s.Fields() {
			if field.IsExported() {
				if !field.IsEmbedded() {
					buffer += (field.Name() + ".")
				}
				a(field.Value(), buffer)
				if !field.IsEmbedded() {
					buffer = buffer[0:strings.LastIndex(buffer, field.Name())]
				}
			}
		}
	}
	a(value, "")
	for _,val := range result {
		if err := viper.BindEnv(val); err != nil {
			return err
			}
	}
	return nil
}
