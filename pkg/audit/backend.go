package audit

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"sync"
)

var (
	auditProcessors []Backend
	mux             = sync.Mutex{}
)

func RegisterProcessor(auditProcessor Backend) {
	mux.Lock()
	defer mux.Unlock()
	auditProcessors = append(auditProcessors, auditProcessor)
}

func Send(events ...*Event) {
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
