package audit

import (
	"encoding/json"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

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
	return append(formatted, []byte("\n")...)
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

func Union(backends ...Backend) Backend {
	if len(backends) == 1 {
		return backends[0]
	}
	return union{backends}
}

type union struct {
	backends []Backend
}

func (u union) Process(events ...*Event) {
	for _, backend := range u.backends {
		backend.Process(events...)
	}
}
