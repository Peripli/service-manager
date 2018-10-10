package testutil

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// LogInterceptor implements logrus.Hook to capture log messages
type LogInterceptor struct {
	strings.Builder
}

// Levels defines which log levels to capture
func (*LogInterceptor) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is called for each log entry
func (hook *LogInterceptor) Fire(entry *logrus.Entry) error {
	str, _ := entry.String()
	hook.WriteString(str)
	return nil
}
