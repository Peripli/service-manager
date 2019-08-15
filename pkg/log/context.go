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

	"github.com/onsi/ginkgo"

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
		"json": &logrus.JSONFormatter{},
		"text": &logrus.TextFormatter{},
	}

	supportedOutputs = map[string]io.Writer{
		os.Stdout.Name(): os.Stdout,
		os.Stdin.Name():  os.Stdin,
		os.Stderr.Name(): os.Stderr,
		"ginkgowriter":   ginkgo.GinkgoWriter,
	}
	mutex           = sync.RWMutex{}
	currentSettings = DefaultSettings()

	once sync.Once

	// C is an alias for ForContext
	C = ForContext
	// D is an alias for Default
	D = Default
)

func init() {
	hook := filename.NewHook()
	hook.Field = FieldComponentName
	defaultEntry.Logger.AddHook(hook)
	defaultEntry = defaultEntry.WithField(FieldCorrelationID, "-")
	//Configure(context.TODO(), currentSettings)
}

// Settings type to be loaded from the environment
type Settings struct {
	Level  string `description:"minimum level for log messages"`
	Format string `description:"format of log messages. Allowed values - text, json"`
	Output string `description:"output for the logs. Allowed values - /dev/stdout, /dev/stdin, /dev/stderr, ginkgowriter"`
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
		return fmt.Errorf("validate Settings: %s", err)
	}
	if len(s.Format) == 0 {
		return fmt.Errorf("validate Settings: LogFormat missing")
	}
	if len(s.Output) == 0 {
		return fmt.Errorf("validate Settings: LogOutput missing")
	}

	return nil
}

// Configure creates a new context with a logger using the provided settings.
func Configure(ctx context.Context, settings *Settings) context.Context {
	mutex.Lock()
	defer mutex.Unlock()

	currentSettings = settings
	level, err := logrus.ParseLevel(settings.Level)
	if err != nil {
		panic(fmt.Sprintf("Could not parse log level configuration: %s", err))
	}
	formatter, ok := supportedFormatters[settings.Format]
	if !ok {
		panic(fmt.Sprintf("Invalid log format: %s", settings.Format))
	}

	output, ok := supportedOutputs[settings.Output]
	if !ok {
		panic(fmt.Sprintf("Invalid output: %s", settings.Output))
	}

	defaultEntry.Logger.SetOutput(output)
	defaultEntry.Logger.SetLevel(level)
	defaultEntry.Logger.SetFormatter(formatter)

	entry := ctx.Value(logKey{})
	if entry == nil {
		entry = defaultEntry
	}

	return ContextWithLogger(ctx, copyEntry(entry.(*logrus.Entry)))
}

func Configuration() *Settings {
	mutex.RLock()
	defer mutex.RUnlock()

	return currentSettings
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
	if _, exists := supportedFormatters[name]; exists {
		return fmt.Errorf("formatter with name %s is already registered", name)
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
	newLogger := &logrus.Logger{
		Out:          entry.Logger.Out,
		Hooks:        entry.Logger.Hooks,
		Formatter:    entry.Logger.Formatter,
		ReportCaller: entry.Logger.ReportCaller,
		Level:        entry.Logger.Level,
		ExitFunc:     entry.Logger.ExitFunc,
	}
	newEntry := logrus.NewEntry(newLogger)
	newEntry.Level = entry.Level
	newEntry.Data = entryData
	newEntry.Time = entry.Time
	newEntry.Message = entry.Message
	newEntry.Buffer = entry.Buffer

	return newEntry
}
