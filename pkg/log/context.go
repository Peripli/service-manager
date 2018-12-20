/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/onrik/logrus/filename"
	"github.com/sirupsen/logrus"
)

type logKey struct{}

var (
	supportedFormatters = map[string]logrus.Formatter{
		"json": &logrus.JSONFormatter{},
		"text": &logrus.TextFormatter{},
	}
	regMutex     = sync.Mutex{}
	once         = sync.Once{}
	defaultEntry = logrus.NewEntry(logrus.StandardLogger())
	// C is an alias for ForContext
	C = ForContext
	// D is an alias for Default
	D = Default
)

const (
	// FieldComponentName is the key of the component field in the log message.
	FieldComponentName = "component"
	// FieldCorrelationID is the key of the correlation id field in the log message.
	FieldCorrelationID = "correlation_id"
)

// Settings type to be loaded from the environment
type Settings struct {
	Level  string
	Format string
}

// DefaultSettings returns default values for Log settings
func DefaultSettings() *Settings {
	return &Settings{
		Level:  "debug",
		Format: "text",
	}
}

// Validate validates the logging settings
func (s *Settings) Validate() error {
	if len(s.Level) == 0 {
		return fmt.Errorf("validate Settings: LogLevel missing")
	}
	if len(s.Format) == 0 {
		return fmt.Errorf("validate Settings: LogFormat missing")
	}
	return nil
}

// Configure creates a new context with a logger using the provided settings.
func Configure(ctx context.Context, settings *Settings) context.Context {
	once.Do(func() {
		level, err := logrus.ParseLevel(settings.Level)
		if err != nil {
			panic(fmt.Sprintf("Could not parse log level configuration: %s", err))
		}
		formatter, ok := supportedFormatters[settings.Format]
		if !ok {
			panic(fmt.Sprintf("Invalid log format: %s", settings.Format))
		}
		logger := &logrus.Logger{
			Formatter: formatter,
			Level:     level,
			Out:       os.Stdout,
			Hooks:     make(logrus.LevelHooks),
		}
		hook := filename.NewHook()
		hook.Field = FieldComponentName
		logger.AddHook(hook)
		defaultEntry = logrus.NewEntry(logger)
		defaultEntry = defaultEntry.WithField(FieldCorrelationID, "-")
	})
	return ContextWithLogger(ctx, defaultEntry)
}

// ForContext retrieves the current logger from the context.
func ForContext(ctx context.Context) *logrus.Entry {
	entry := ctx.Value(logKey{})
	if entry == nil {
		entry = defaultEntry
	}
	return copyEntry(entry.(*logrus.Entry))
}

// Default returns the default logger
func Default() *logrus.Entry {
	return ForContext(context.Background())
}

// ContextWithLogger returns a new context with the provided logger.
func ContextWithLogger(ctx context.Context, entry *logrus.Entry) context.Context {
	return context.WithValue(ctx, logKey{}, entry)
}

// RegisterFormatter registers a new logrus Formatter with the given name.
// Returns an error if there is a formatter with the same name.
func RegisterFormatter(name string, formatter logrus.Formatter) error {
	regMutex.Lock()
	defer regMutex.Unlock()
	if _, exists := supportedFormatters[name]; exists {
		return fmt.Errorf("Formatter with name %s is already registered", name)
	}
	supportedFormatters[name] = formatter
	return nil
}

// AddHook adds a hook to all loggers
func AddHook(hook logrus.Hook) {
	defaultEntry.Logger.AddHook(hook)
}

func copyEntry(entry *logrus.Entry) *logrus.Entry {
	entryData := make(logrus.Fields, len(entry.Data))
	for k, v := range entry.Data {
		entryData[k] = v
	}
	return &logrus.Entry{
		Logger: &logrus.Logger{
			Level:     entry.Logger.Level,
			Formatter: entry.Logger.Formatter,
			Hooks:     entry.Logger.Hooks,
			Out:       entry.Logger.Out,
		},
		Level:   entry.Level,
		Data:    entryData,
		Time:    entry.Time,
		Message: entry.Message,
		Buffer:  entry.Buffer,
	}
}
