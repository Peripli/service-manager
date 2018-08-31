package filters

import (
	"github.com/Peripli/service-manager/pkg/audit"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
)

type AuditFilter struct {
	auditBackend audit.Backend
}

func (*AuditFilter) Name() string {
	return "AuditFilter"
}

func (f *AuditFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	event, e := audit.NewEventForRequest(req)
	if e != nil {
		return nil, e
	}

	event.State = audit.StatePreRequest
	f.auditBackend.Process(event)

	resp, err := next.Handle(req)
	if err != nil {
		event.State = audit.StateError
		event.ResponseStatus = http.StatusInternalServerError
		event.ResponseError = err
	} else {
		event.State = audit.StatePostRequest
		event.ResponseStatus = resp.StatusCode
		event.ResponseObject = resp.Body
	}
	f.auditBackend.Process(event)
	return resp, err
}

func (*AuditFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
			},
		},
	}
}
