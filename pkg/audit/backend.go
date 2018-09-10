package audit

import (
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	auditProcessors []Backend
	regMutex        = sync.Mutex{}
)

func RegisterProcessor(auditProcessor Backend) {
	regMutex.Lock()
	defer regMutex.Unlock()
	auditProcessors = append(auditProcessors, auditProcessor)
}

func Process(events ...*Event) {
	for _, processor := range auditProcessors {
		processor.Process(events...)
	}
}

func NewStdoutJsonBackend() Backend {
	return &WriterBackend{
		Formatter: &JsonFormatter{},
		Writer:    os.Stdout,
	}
}

type JsonFormatter struct {
}

func (*JsonFormatter) Format(event *Event) []byte {
	formatted, err := json.Marshal(event)
	if err != nil {
		logrus.Errorf("Could not create formatted audit event. Affected event: %v", event)
		return []byte{}
	}
	return formatted
}

type WriterBackend struct {
	Formatter Formatter
	Writer    io.Writer
}

func (b *WriterBackend) Process(events ...*Event) {
	for _, e := range events {
		formattedEvent := b.Formatter.Format(e)
		b.Writer.Write(formattedEvent)
	}
}
