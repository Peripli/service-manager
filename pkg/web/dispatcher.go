package web

import (
	"strings"
)

// DispatchingFilter represents a Named Middleware that can participate in a Dispatcher.
type DispatchingFilter interface {
	Named
	Middleware

	Matches(request *Request) bool
}

// Dispatcher dispatches the request to a subset of the provided Middlewares based on dynamic request matching.
// It is a Named Middleware that matches a set of filters against a Request and executes the ones that match.
type Dispatcher struct {
	Filters []DispatchingFilter
}

func (d Dispatcher) Run(next Handler) Handler {
	return HandlerFunc(func(request *Request) (*Response, error) {
		fs := Middlewares{}
		for _, filter := range d.Filters {
			if filter.Matches(request) {
				fs = append(fs, filter)
			}
		}
		return fs.Chain(next).Handle(request)
	})
}

func (d Dispatcher) Name() string {
	s := []string{"Dispatcher: "}
	for _, filter := range d.Filters {
		s = append(s, filter.Name())
	}
	return strings.Join(s, "<->")
}
