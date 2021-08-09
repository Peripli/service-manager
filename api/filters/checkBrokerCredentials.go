package filters

import (
	"errors"
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
	smBrokerCredentials := gjson.GetBytes(req.Body, "sm_provided_credentials")
	basicFields := gjson.GetManyBytes(req.Body, fmt.Sprintf(basicCredentialsPath, "username"), fmt.Sprintf(basicCredentialsPath, "password"))
	tlsFields := gjson.GetManyBytes(req.Body, fmt.Sprintf(tlsCredentialsPath, "client_certificate"), fmt.Sprintf(tlsCredentialsPath, "client_key"))
	err := credentialsMissing(smBrokerCredentials, basicFields, tlsFields)
	if brokerUrl.Exists() && err != nil {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: err.Error(),
			StatusCode:  http.StatusBadRequest,
		}
	}
	return next.Handle(req)
}

func credentialsMissing(smBrokerCredentials gjson.Result, basicFields []gjson.Result, tlsFields []gjson.Result) error {
	httpSettings := httpclient.GetHttpClientGlobalSettings()
	if smBrokerCredentials.Exists() && smBrokerCredentials.Bool() == true && len(httpSettings.ServerCertificate) == 0 {
		return errors.New("No SM provided credentials available, provide another type of credentials")
	}
	if (smBrokerCredentials.Exists() == true && smBrokerCredentials.Bool() == true || basicFields[0].Exists() && basicFields[1].Exists()) || (tlsFields[0].Exists() && tlsFields[1].Exists()) {
		return nil
	}
	return errors.New("Updating a URL of a broker requires its credentials")
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
