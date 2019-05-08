package testutil

import (
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// LogInterceptor implements logrus.Hook to capture log messages
type LogInterceptor struct {
	strings.Builder
	bufferMutex sync.Mutex
}

// Levels defines which log levels to capture
func (*LogInterceptor) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is called for each log entry
func (li *LogInterceptor) Fire(entry *logrus.Entry) error {
	str, err := entry.String()
	if err != nil {
		return err
	}
	li.bufferMutex.Lock()
	defer li.bufferMutex.Unlock()
	_, err = li.WriteString(str)
	return err
}

func (li *LogInterceptor) String() string {
	li.bufferMutex.Lock()
	defer li.bufferMutex.Unlock()
	return li.Builder.String()
}
