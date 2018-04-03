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

	"github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

func init() {
	if err := Load("./config"); err != nil {
		panic(fmt.Sprintf("Could not load environment configuration: %v", err))
	}
}

// Get returns the value associated with this key from the environment
func Get(key string) interface{} {
	return viper.Get(key)
}

// Load loads the application.yml file from the specified location
func Load(location string) error {
	viper.AddConfigPath(location)
	viper.SetConfigName("application")
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("Could not read configuration file: %s", err))
	}
	initializeLogging()
	return nil
}

func initializeLogging() {
	logLevel := viper.GetString("log.level")
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.WithField("level", level).Warn("Missing or invalid log level! Falling back to Error...")
		logrus.SetLevel(logrus.ErrorLevel)
	} else {
		logrus.SetLevel(level)
	}
}
