package filters

import (
	"errors"
	"fmt"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/httpclient"
	"net/http"

	"github.com/tidwall/gjson"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
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
	smBrokerCredentials := gjson.GetBytes(req.Body, fmt.Sprintf(tlsCredentialsPath, "sm_provided_tls_credentials"))
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
	if smBrokerCredentials.Exists() && smBrokerCredentials.Bool() && len(httpSettings.ServerCertificate) == 0 {
		return errors.New("no sm provided credentials available, provide another type of credentials")
	}
	smProvided := smBrokerCredentials.Exists() && smBrokerCredentials.Bool()
	basic := basicFields[0].Exists() && basicFields[1].Exists()
	tls := tlsFields[0].Exists() && tlsFields[1].Exists()
	if tls || basic || smProvided {
		return nil
	}
	return errors.New("updating a url of a broker requires its credentials")
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
