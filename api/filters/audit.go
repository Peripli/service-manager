package filters

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/audit"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
)

type AuditFilter struct {
}

func (*AuditFilter) Name() string {
	return "AuditFilter"
}

func (f *AuditFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	req, event, e := audit.RequestWithEvent(req)
	if e != nil {
		return nil, fmt.Errorf("failed to create audit event: %v", err)
	}

	event.State = audit.StatePreRequest
	audit.Send(event)

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
	audit.Send(event)
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
