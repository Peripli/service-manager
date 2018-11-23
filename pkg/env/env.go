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
	"os"
	"reflect"
	"strings"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/fatih/structs"
	"github.com/spf13/cast"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// File describes the name, path and the format of the file to be used to load the configuration in the env
type File struct {
	Name     string `description:"name of the configuration file"`
	Location string `description:"location of the configuration file"`
	Format   string `description:"extension of the configuration file"`
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
	return set
}

// CreatePFlags Creates pflags for the value structure and adds them in the provided set
func CreatePFlags(set *pflag.FlagSet, value interface{}) {
	tree := &DescriptionTree{}
	var parameters []Parameter
	s := structs.New(value)
	for _, field := range s.Fields() {
		if field.Tag("description") != "" {
			baseTree := NewDescriptionTree(field.Tag("description"))
			buildFlagDescriptionTree(field, baseTree, "", &parameters)
		}
	}
	buildFlagDescriptionTree(value, tree, "", &parameters)
	descriptions := buildFlagDescriptionsFromTree(tree)
	descriptionsCount := len(descriptions)
	parametersCount := len(parameters)
	if descriptionsCount < parametersCount {
		log.D().Warnf("Unexpected number of descriptions found for %s: %d. Expected the same number as the configuration parameters: %d. Using default descriptions...", s.Names(), descriptionsCount, parametersCount)
		for _, binding := range parameters {
			descriptions = append(descriptions, fmt.Sprintf("commandline argument for %s", binding.Name))
		}
	}

	for i, bindingName := range parameters {
		if set.Lookup(bindingName.Name) == nil {
			set.String(bindingName.Name, bindingName.DefaultValue, descriptions[i])
		}
	}
}

func buildFlagDescriptionsFromTree(tree *DescriptionTree) []string {
	return buildDescriptionPaths(tree, []*DescriptionTree{})
}

func buildDescriptionPaths(root *DescriptionTree, path []*DescriptionTree) []string {
	path = append(path, root)
	var result []string
	if root.Children == nil || len(root.Children) == 0 {
		res := ""
		for _, node := range path {
			res += node.Value
		}
		if res != "" {
			result = append(result, res)
		}
		return result
	}
	for _, node := range root.Children {
		result = append(result, buildDescriptionPaths(node, path)...)
	}
	return result
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
			log.D().Panic(err)
		}
	})

	if err := v.setupConfigFile(); err != nil {
		return nil, err
	}

	return v, nil
}

type DescriptionTree struct {
	Value    string
	Children []*DescriptionTree
}

func NewDescriptionTree(root string) *DescriptionTree {
	return &DescriptionTree{
		Value:    root,
		Children: nil,
	}
}

func (t *DescriptionTree) AddNode(tree *DescriptionTree) {
	if t.Children == nil {
		t.Children = []*DescriptionTree{tree}
		return
	}
	t.Children = append(t.Children, tree)
}

type Parameter struct {
	Name         string
	DefaultValue string
}

// Unmarshal exposes viper's Unmarshal. Prior to unmarshaling it creates the necessary pflag and env var bindings
// so that pflag / env var values are also used during the unmarshaling.
func (v *ViperEnv) Unmarshal(value interface{}) error {
	tree := &DescriptionTree{}
	var parameters []Parameter
	s := structs.New(value)
	for _, field := range s.Fields() {
		if field.Tag("description") != "" {
			baseTree := NewDescriptionTree(field.Tag("description"))
			buildFlagDescriptionTree(field, baseTree, "", &parameters)
		}
	}
	buildFlagDescriptionTree(value, tree, "", &parameters)
	for _, bindingName := range parameters {
		if err := v.Viper.BindEnv(bindingName.Name); err != nil {
			return err
		}
	}

	return v.Viper.Unmarshal(value)
}

func buildFlagDescriptionTree(value interface{}, tree *DescriptionTree, buffer string, result *[]Parameter) {
	if !structs.IsStruct(value) {
		index := strings.LastIndex(buffer, ".")
		if index == -1 {
			index = 0
		}
		key := strings.ToLower(buffer[0:index])
		*result = append(*result, Parameter{Name: key, DefaultValue: cast.ToString(value)})
		tree.Children = nil
		return
	}
	var fields []*structs.Field
	s := structs.New(value)
	for _, field := range s.Fields() {
		if field.Kind() != reflect.Slice {
			fields = append(fields, field)
		}
	}
	for i, field := range fields {
		var name string
		if field.Tag("mapstructure") != "" {
			name = field.Tag("mapstructure")
		} else {
			name = field.Name()
		}
		description := ""
		if field.Tag("description") != "" {
			description = field.Tag("description")
		}

		baseTree := NewDescriptionTree(description)
		tree.AddNode(baseTree)
		buffer += name + "."
		buildFlagDescriptionTree(field.Value(), tree.Children[i], buffer, result)
		buffer = buffer[0:strings.LastIndex(buffer, name)]
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
			log.D().Info("Config File was not found: ", err)
			return nil
		}
		return fmt.Errorf("could not read configuration cfg: %s", err)
	}
	return nil
}
