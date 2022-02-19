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
	"io"
	"os"
	"sync"

	"github.com/onsi/ginkgo/v2"

	"github.com/onrik/logrus/filename"

	"github.com/sirupsen/logrus"
)

type logKey struct{}

const (
	// FieldComponentName is the key of the component field in the log message.
	FieldComponentName = "component"
	// FieldCorrelationID is the key of the correlation id field in the log message.
	FieldCorrelationID = "correlation_id"
)

var (
	defaultEntry = logrus.NewEntry(logrus.StandardLogger())

	supportedFormatters = map[string]logrus.Formatter{
		"json":   &logrus.JSONFormatter{},
		"text":   &logrus.TextFormatter{},
		"kibana": &KibanaFormatter{},
	}

	supportedOutputs = map[string]io.Writer{
		os.Stdout.Name(): os.Stdout,
		os.Stderr.Name(): os.Stderr,
		"ginkgowriter":   ginkgo.GinkgoWriter,
	}
	mutex           = sync.RWMutex{}
	currentSettings = DefaultSettings()

	// C is an alias for ForContext
	C = ForContext
	// D is an alias for DefaultLogger
	D = DefaultLogger
)

func init() {
	hook := filename.NewHook()
	hook.Field = FieldComponentName
	defaultEntry.Logger.AddHook(hook)
	defaultEntry = defaultEntry.WithField(FieldCorrelationID, "-")
	_, err := Configure(context.Background(), currentSettings)
	if err != nil {
		panic(err)
	}
}

// Settings type to be loaded from the environment
type Settings struct {
	Level  string `description:"minimum level for log messages" json:"level,omitempty"`
	Format string `description:"format of log messages. Allowed values - text, json" json:"format,omitempty"`
	Output string `description:"output for the logs. Allowed values - /dev/stdout, /dev/stderr, ginkgowriter" json:"output,omitempty"`
}

// DefaultSettings returns default values for Log settings
func DefaultSettings() *Settings {
	return &Settings{
		Level:  "error",
		Format: "text",
		Output: os.Stdout.Name(),
	}
}

// Validate validates the logging settings
func (s *Settings) Validate() error {
	if _, err := logrus.ParseLevel(s.Level); err != nil {
		return fmt.Errorf("validate Settings: log level %s is invalid: %s", s.Level, err)
	}

	if len(s.Format) == 0 {
		return fmt.Errorf("validate Settings: log format missing")
	}

	if _, ok := supportedFormatters[s.Format]; !ok {
		return fmt.Errorf("validate Settings: log format %s is invalid", s.Format)
	}

	if len(s.Output) == 0 {
		return fmt.Errorf("validate Settings: log output missing")
	}

	if _, ok := supportedOutputs[s.Output]; !ok {
		return fmt.Errorf("validate Settings: log output %s is invalid", s.Output)
	}

	return nil
}

// Configure creates a new context with a logger using the provided settings.
func Configure(ctx context.Context, settings *Settings) (context.Context, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if err := settings.Validate(); err != nil {
		return nil, err
	}

	level, _ := logrus.ParseLevel(settings.Level)
	formatter := supportedFormatters[settings.Format]
	output := supportedOutputs[settings.Output]

	currentSettings = settings

	entry := ctx.Value(logKey{})
	if entry == nil {
		entry = defaultEntry
	} else {
		defaultEntry.Logger.SetOutput(output)
		defaultEntry.Logger.SetFormatter(formatter)
		defaultEntry.Logger.SetLevel(level)
	}
	e := copyEntry(entry.(*logrus.Entry))
	e.Logger.SetLevel(level)
	e.Logger.SetFormatter(formatter)
	e.Logger.SetOutput(output)

	return ContextWithLogger(ctx, e), nil
}

// Configuration returns the logger settings
func Configuration() Settings {
	mutex.RLock()
	defer mutex.RUnlock()

	return *currentSettings
}

// ForContext retrieves the current logger from the context.
func ForContext(ctx context.Context) *logrus.Entry {
	mutex.RLock()
	defer mutex.RUnlock()
	entry := ctx.Value(logKey{})
	if entry == nil {
		entry = defaultEntry
	}
	return copyEntry(entry.(*logrus.Entry))
}

// DefaultLogger returns the default logger
func DefaultLogger() *logrus.Entry {
	return ForContext(context.Background())
}

// ContextWithLogger returns a new context with the provided logger.
func ContextWithLogger(ctx context.Context, entry *logrus.Entry) context.Context {
	return context.WithValue(ctx, logKey{}, entry)
}

// RegisterFormatter registers a new logrus Formatter with the given name.
// Returns an error if there is a formatter with the same name.
func RegisterFormatter(name string, formatter logrus.Formatter) error {
	if _, exists := supportedFormatters[name]; exists {
		return fmt.Errorf("formatter with name %s is already registered", name)
	}
	supportedFormatters[name] = formatter
	return nil
}

// CorrelationIDFromContext returns the correlation id associated with the context logger or empty string if none exists
func CorrelationIDFromContext(ctx context.Context) string {
	correlationID, exists := C(ctx).Data[FieldCorrelationID]
	if exists {
		if id, ok := correlationID.(string); ok {
			return id
		}
	}
	return ""
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

	newEntry := logrus.NewEntry(entry.Logger)
	newEntry.Level = entry.Level
	newEntry.Data = entryData
	newEntry.Time = entry.Time
	newEntry.Message = entry.Message
	newEntry.Buffer = entry.Buffer

	return newEntry
}
