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

	"github.com/sirupsen/logrus"
)

type logKey struct{}

var (
	supportedFormatters = map[string]logrus.Formatter{
		"json": &logrus.JSONFormatter{},
		"text": &logrus.TextFormatter{},
	}
	mux                       = sync.Mutex{}
	defaultEntry              = logrus.NewEntry(logrus.StandardLogger())
	initializationError error = nil
	once                      = sync.Once{}
	C                         = ForContext
	R                         = ForContextProvider
	D                         = Default
)

type Contexter interface {
	Context() context.Context
}

func Configure(ctx context.Context, settings *Settings) (context.Context, error) {
	once.Do(func() {
		level, err := logrus.ParseLevel(settings.Level)
		if err != nil {
			initializationError = fmt.Errorf("Could not parse log level configuration: %s", err)
			return
		}
		formatter, ok := supportedFormatters[settings.Format]
		if !ok {
			initializationError = fmt.Errorf("Invalid log format: %s", settings.Format)
			return
		}
		logrus.SetLevel(level)
		logrus.SetFormatter(formatter)
		logger := &logrus.Logger{
			Formatter: formatter,
			Level:     level,
			Out:       os.Stdout,
			Hooks:     make(logrus.LevelHooks),
		}
		defaultEntry = logrus.NewEntry(logger)
	})
	return ContextWithLogger(ctx, defaultEntry), initializationError
}

func ForContext(ctx context.Context, component string) *logrus.Entry {
	entry := ctx.Value(logKey{})
	if entry == nil {
		entry = defaultEntry
	}
	return entry.(*logrus.Entry).WithField("package", component)
}

func ForContextProvider(contexter Contexter, component string) *logrus.Entry {
	return ForContext(contexter.Context(), component)
}

func Default(component string) *logrus.Entry {
	return ForContext(context.Background(), component)
}

func ContextWithLogger(ctx context.Context, entry *logrus.Entry) context.Context {
	return context.WithValue(ctx, logKey{}, entry)
}

func RegisterFormatter(name string, formatter logrus.Formatter) {
	mux.Lock()
	defer mux.Unlock()
	supportedFormatters[name] = formatter
}
