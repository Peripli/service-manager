package filters

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/tidwall/gjson"
)

const (
	CheckBrokerCredentialsFilterName = "CheckBrokerCredentialsFilter"
	credentialsPath                  = "credentials.basic.%s"
)

// CheckBrokerCredentialsFilter checks patch request for the broker basic credentials
type CheckBrokerCredentialsFilter struct {
}

func (*CheckBrokerCredentialsFilter) Name() string {
	return CheckBrokerCredentialsFilterName
}

func (*CheckBrokerCredentialsFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	fields := gjson.GetManyBytes(req.Body, "broker_url", fmt.Sprintf(credentialsPath, "username"), fmt.Sprintf(credentialsPath, "password"))

	if fields[0].Exists() && (!fields[1].Exists() || !fields[2].Exists()) {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "Updating an URL of a broker requires its basic credentials",
			StatusCode:  http.StatusBadRequest,
		}
	}
	return next.Handle(req)
}

func (*CheckBrokerCredentialsFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBrokersURL + "/**"),
				web.Methods(http.MethodPatch),
			},
		},
	}
}
