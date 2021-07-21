package filters

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/httpclient"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/tidwall/gjson"
)

const (
	CheckBrokerCredentialsFilterName = "CheckBrokerCredentialsFilter"
	basicCredentialsPath             = "credentials.basic.%s"
	tlsCredentialsPath               = "credentials.tls.%s"
)

// CheckBrokerCredentialsFilter checks patch request for the broker basic credentials
type CheckBrokerCredentialsFilter struct {
}

func (*CheckBrokerCredentialsFilter) Name() string {
	return CheckBrokerCredentialsFilterName
}

func (*CheckBrokerCredentialsFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	brokerUrl := gjson.GetBytes(req.Body, "broker_url")
	basicFields := gjson.GetManyBytes(req.Body, fmt.Sprintf(basicCredentialsPath, "username"), fmt.Sprintf(basicCredentialsPath, "password"))
	tlsFields := gjson.GetManyBytes(req.Body, fmt.Sprintf(tlsCredentialsPath, "client_certificate"), fmt.Sprintf(tlsCredentialsPath, "client_key"))

	if brokerUrl.Exists() && credentialsMissing(basicFields, tlsFields) {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "Updating a URL of a broker requires its credentials",
			StatusCode:  http.StatusBadRequest,
		}
	}
	return next.Handle(req)
}

func credentialsMissing(basicFields []gjson.Result, tlsFields []gjson.Result) bool {
	httpSettings := httpclient.GetHttpClientGlobalSettings()
	if (basicFields[0].Exists() && basicFields[1].Exists()) || (tlsFields[0].Exists() && tlsFields[1].Exists()) || len(httpSettings.ServerCertificate) > 0 {
		return false
	}
	return true
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
