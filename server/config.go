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

package server

import (
	"strconv"
	"time"
)

type Environment interface {
	Load() error
	Get(key string) interface{}
	Unmarshal(value interface{}) error
}

type Settings struct {
	Server serverSettings
	Db     dbSettings
	Log    logSettings
}

type serverSettings struct {
	Port            int
	RequestTimeout  int
	ShutdownTimeout int
}

type dbSettings struct {
	URI string
}

type logSettings struct {
	Level  string
	Format string
}

type Config struct {
	Address         string
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
	LogLevel        string
	LogFormat       string
	DbURI           string
}

func NewConfiguration(env Environment) (*Config, error) {
	config := DefaultConfiguration()

	if err := env.Load(); err != nil {
		return nil, err
	}

	configSettings := &Settings{}
	if err := env.Unmarshal(configSettings); err != nil {
		return nil, err
	}

	if configSettings.Server.Port != 0 {
		config.Address = ":" + strconv.Itoa(configSettings.Server.Port)
	}
	if configSettings.Server.RequestTimeout != 0 {
		config.RequestTimeout = time.Duration(configSettings.Server.RequestTimeout)
	}
	if configSettings.Server.ShutdownTimeout != 0 {
		config.ShutdownTimeout = time.Duration(configSettings.Server.ShutdownTimeout)
	}
	if len(configSettings.Db.URI) != 0 {
		config.DbURI = configSettings.Db.URI
	}
	if len(configSettings.Log.Format) != 0 {
		config.LogFormat = configSettings.Log.Format
	}
	if len(configSettings.Log.Level) != 0 {
		config.LogLevel = configSettings.Log.Level
	}

	return config, nil
}

// DefaultConfiguration returns a default server configuration
func DefaultConfiguration() *Config {
	config := &Config{
		Address:         ":8080",
		RequestTimeout:  time.Millisecond * time.Duration(3000),
		ShutdownTimeout: time.Millisecond * time.Duration(3000),
		LogLevel:        "debug",
		LogFormat:       "text",
		DbURI:           "",
	}

	return config
}
