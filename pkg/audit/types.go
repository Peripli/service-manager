package audit

import "time"

type State string

const (
	StatePreRequest  State = "PreRequest"
	StatePostRequest State = "PostRequest"
	StateError       State = "RequestError"
)

type Event struct {
	AuditID string

	Verb      string
	UserAgent string
	SourceIPs []string

	RequestURI       string
	RequestTimestamp time.Time
	RequestObject    []byte

	ResponseStatus int
	ResponseObject []byte
	ResponseError  error

	State State

	Metadata map[string]string
}

type Backend interface {
	Process(events ...*Event)
}

type Formatter interface {
	Format(event *Event) []byte
}
