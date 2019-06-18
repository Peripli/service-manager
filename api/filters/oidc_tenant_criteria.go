package filters

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util/slice"

	"github.com/tidwall/gjson"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/web"
)

//TODO
// add some component tests that sm delete/list/patch respond in tenant aware manner - would be good if those are part of the generic tests
// sm-dev adopt and add new config properties in the dev en
// OIDCTenantCriteriaFilterName is the name of the filter
const OIDCTenantCriteriaFilterName = "OIDCTenantCriteriaFilter"

// TenantCriteriaSettings are the settings required to configure the OIDCTenantCriteriaFilter
type TenantCriteriaSettings struct {
	ClientID           []string `mapstructure:"client_id" description:"id of the client from which the token should be issued in order to enable tenant resource filtering"`
	LabelKey           string   `mapstructure:"label_key" description:"the name of the label by the value of which resources will be filtered"`
	ClientIDTokenClaim string   `mapstructure:"client_id_token_claim" description:"the name of the claim in the OIDC token that contains the client ID"`
	LabelTokenClaim    string   `mapstructure:"label_token_claim" description:"the name of the claim in the OIDC token that contains the value of the label key that will be used for filtering the resources"`
}

// NewOIDCTenantCriteriaFilter creates a new OIDCTenantCriteriaFilter from the specified settings
func NewOIDCTenantCriteriaFilter(settings *TenantCriteriaSettings) *OIDCTenantCriteriaFilter {
	return &OIDCTenantCriteriaFilter{
		ClientIDs:          settings.ClientID,
		ClientIDTokenClaim: settings.ClientIDTokenClaim,
		LabelKey:           settings.LabelKey,
		LabelTokenClaim:    settings.LabelTokenClaim,
	}
}

// OIDCTenantCriteriaFilter filters the resources on GET, DELETE and PATCH during OAuth authentication based on the specified parameters
type OIDCTenantCriteriaFilter struct {
	// ClientIDs are the client IDs of the OAuth clients, the tokens of which are issued for users with multi tenant access
	ClientIDs []string
	// ClientIDTokenClaim is the token claim that contains the value of the client ID of the client from which the token was issued
	ClientIDTokenClaim string
	// LabelKey is the key of the label, the value of which will be used to filter the returned resources for this user
	LabelKey string
	// LabelTokenClaim is the token claim that contains the LabelKey value for the user for which the token was issued
	LabelTokenClaim string
}

// Name implements web.Named and returns the filter name
func (f *OIDCTenantCriteriaFilter) Name() string {
	return OIDCTenantCriteriaFilterName
}

// Run implements web.Middleware and attempts to add additional label crieria to the context that would be used to filter the resources
func (f *OIDCTenantCriteriaFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	logger := log.C(ctx)
	if len(f.ClientIDs) == 0 || f.ClientIDTokenClaim == "" || f.LabelKey == "" || f.LabelTokenClaim == "" {
		logger.Infof("Tenant criteria filtering is not configured. Proceeding with next filter...")
		return next.Handle(request)
	}

	user, ok := web.UserFromContext(ctx)
	if !ok {
		logger.Infof("No user found in user context. Proceeding with next filter...")
		return next.Handle(request)
	}

	if user.AuthenticationType != web.Bearer {
		logger.Infof("Authentication type is not Bearer. Proceeding with next filter...")
		return next.Handle(request)
	}

	var userData json.RawMessage
	if err := user.DataFunc(&userData); err != nil {
		return nil, fmt.Errorf("could not unmarshal claims from token: %s", err)
	}

	clientIDFromToken := gjson.GetBytes([]byte(userData), f.ClientIDTokenClaim).String()
	if slice.StringsAnyEquals(f.ClientIDs, clientIDFromToken) {
		logger.Infof("Token in user context was issued by %s and not by the tenant aware client %s. Proceeding with next filter...", clientIDFromToken, f.ClientIDs)
		return next.Handle(request)
	}

	delimiterClaimValue := gjson.GetBytes([]byte(userData), f.LabelTokenClaim).String()
	if len(delimiterClaimValue) == 0 {
		return nil, fmt.Errorf("invalid token: could not find delimiter %s in token claims", f.LabelTokenClaim)
	}

	criterion := query.ByLabel(query.EqualsOperator, f.LabelKey, delimiterClaimValue)
	var err error
	ctx, err = query.AddCriteria(ctx, criterion)
	if err != nil {
		return nil, fmt.Errorf("could not add label critaria with key %s and value %s: %s", f.LabelKey, delimiterClaimValue, err)
	}

	request.Request = request.WithContext(ctx)
	logger.Infof("Successfully added label criteria with key %s and value %s to context", f.LabelKey, delimiterClaimValue)

	return next.Handle(request)
}

// FilterMatchers implements web.Filter.FilterMatchers and specifies that the filter should run on GET, PATCH and DELETE for all resources
func (*OIDCTenantCriteriaFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet, http.MethodPatch, http.MethodDelete),
			},
		},
	}
}
