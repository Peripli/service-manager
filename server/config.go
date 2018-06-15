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
	"time"

	"github.com/spf13/pflag"
)

func init() {
	pflag.String("db_uri", "", "Database URI used to connect to SM DB")
	//IntroducePFlags(Config{})
}

// AppSettings type to be loaded from the environment
type AppSettings struct {
	Host            string
	Port            int
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
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

//TODO we could add the file and tokenissuer configs in here too, so we have to init pflag only here?
// Config type to be loaded from the environment
type Config struct {
	Server AppSettings
	DB     DbSettings
	Log    LogSettings
}

func DefaultConfig() *Config {
	config := &Config{
		Server: AppSettings{
			Host:            "127.0.0.1",
			Port:            8080,
			RequestTimeout:  time.Millisecond * time.Duration(3000),
			ShutdownTimeout: time.Millisecond * time.Duration(3000),
		},
		DB: DbSettings{
			URI: "",
		},
		Log: LogSettings{
			Level:  "debug",
			Format: "text",
		},
	}
	return config
}

// NewConfig creates a configuration from the provided environment
func NewConfig(env Environment) (*Config, error) {
	config := DefaultConfig()

	//TODO we could do the below but then we need to move the info details tokenissuer as part of the config and pass config to the info controller instead of environment
	//if err := env.Introduce(config); err != nil {
	//	return nil, err
	//}
	if err := env.Load(); err != nil {
		return nil, err
	}
	if err := env.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates that the configuration contains all mandatory properties
func (c *Config) Validate() error {
	if c.Server.Port == 0 {
		return fmt.Errorf("validate Config: Port missing")
	}
	if c.Server.RequestTimeout == 0 {
		return fmt.Errorf("validate Config: RequestTimeout missing")
	}
	if c.Server.ShutdownTimeout == 0 {
		return fmt.Errorf("validate Config: ShutdownTimeout missing")
	}
	if len(c.Log.Level) == 0 {
		return fmt.Errorf("validate Config: LogLevel missing")
	}
	if len(c.Log.Format) == 0 {
		return fmt.Errorf("validate Config: LogFormat missing")
	}
	if len(c.DB.URI) == 0 {
		return fmt.Errorf("validate Config: DbURI missing")
	}
	return nil
}
