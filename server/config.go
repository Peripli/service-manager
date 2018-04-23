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
	"fmt"
	"strconv"
	"time"
)

// Environment represents an abstraction over the environment from which Service Manager configuration will be loaded
//go:generate counterfeiter . Environment
type Environment interface {
	Load() error
	Get(key string) interface{}
	Unmarshal(value interface{}) error
}

// Settings type to be loaded from the environment
type Settings struct {
	Server *AppSettings
	Db     *DbSettings
	Log    *LogSettings
}

// AppSettings type to be loaded from the environment
type AppSettings struct {
	Port            int
	RequestTimeout  int
	ShutdownTimeout int
}

// DbSettings type to be loaded from the environment
type DbSettings struct {
	URI string
}

// LogSettings type to be loaded from the environment
type LogSettings struct {
	Level  string
	Format string
}

// Config type represents Service Manager configuration
type Config struct {
	Address         string
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
	LogLevel        string
	LogFormat       string
	DbURI           string
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

// NewConfiguration creates a configuration from the provided environment
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
		config.RequestTimeout = time.Millisecond * time.Duration(configSettings.Server.RequestTimeout)
	}
	if configSettings.Server.ShutdownTimeout != 0 {
		config.ShutdownTimeout = time.Millisecond * time.Duration(configSettings.Server.ShutdownTimeout)
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

// Validate validates that the configuration contains all mandatory properties
func (c *Config) Validate() error {
	if len(c.Address) == 0 {
		return fmt.Errorf("validate Config: Address missing")
	}
	if c.RequestTimeout == 0 {
		return fmt.Errorf("validate Config: RequestTimeout missing")
	}
	if c.ShutdownTimeout == 0 {
		return fmt.Errorf("validate Config: ShutdownTimeout missing")
	}
	if len(c.LogLevel) == 0 {
		return fmt.Errorf("validate Config: LogLevel missing")
	}
	if len(c.LogFormat) == 0 {
		return fmt.Errorf("validate Config: LogFormat missing")
	}
	if len(c.DbURI) == 0 {
		return fmt.Errorf("validate Config: DbURI missing")
	}
	return nil
}
