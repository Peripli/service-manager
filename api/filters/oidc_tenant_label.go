package filters

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/tidwall/gjson"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/web"
)

// sm-dev adopt and add new config properties in the dev en
const OIDCTenantCriteriaFilterName = "OIDCTenantCriteriaFilter"
const OIDCTenantLabelingFilterName = "OIDCTenantLabelingFilter"

// TenantCriteriaSettings are the settings required to configure the OIDCTenantCriteriaFilter
type TenantCriteriaSettings struct {
	ClientID           string `mapstructure:"client_id" description:"id of the client from which the token should be issued in order to enable tenant resource filtering"`
	LabelKey           string `mapstructure:"label_key" description:"the name of the label by the value of which resources will be filtered"`
	ClientIDTokenClaim string `mapstructure:"client_id_token_claim" description:"the name of the claim in the OIDC token that contains the client ID"`
	LabelTokenClaim    string `mapstructure:"label_token_claim" description:"the name of the claim in the OIDC token that contains the value of the label key that will be used for filtering the resources"`
}

// NewOIDCTenantCriteriaFilter creates a new OIDCTenantFilter from the specified settings that filters the returned resources based on a filtering label
func NewOIDCTenantCriteriaFilter(settings *TenantCriteriaSettings) *OIDCTenantFilter {
	return &OIDCTenantFilter{
		ClientID:           settings.ClientID,
		ClientIDTokenClaim: settings.ClientIDTokenClaim,
		LabelKey:           settings.LabelKey,
		LabelTokenClaim:    settings.LabelTokenClaim,
		FilterName:         OIDCTenantCriteriaFilterName,
		Methods:            []string{http.MethodGet, http.MethodPatch, http.MethodDelete},
		ProcessFunc: func(request *web.Request, labelKey, labelValue string) error {
			ctx := request.Context()
			criterion := query.ByLabel(query.EqualsOperator, labelKey, labelValue)
			var err error
			ctx, err = query.AddCriteria(ctx, criterion)
			if err != nil {
				return fmt.Errorf("could not add label critaria with key %s and value %s: %s", labelKey, labelValue, err)
			}

			log.C(ctx).Infof("Successfully added label criteria with key %s and value %s to context", labelKey, labelValue)
			request.Request = request.WithContext(ctx)

			return nil
		},
	}
}

// NewOIDCTenantLabelingFilter creates a new OIDCTenantFilter from the specified settings that adds a filtering label when creating resources
func NewOIDCTenantLabelingFilter(settings *TenantCriteriaSettings) *OIDCTenantFilter {
	return &OIDCTenantFilter{
		ClientID:           settings.ClientID,
		ClientIDTokenClaim: settings.ClientIDTokenClaim,
		LabelKey:           settings.LabelKey,
		LabelTokenClaim:    settings.LabelTokenClaim,
		FilterName:         OIDCTenantLabelingFilterName,
		Methods:            []string{http.MethodPost},
		ProcessFunc: func(request *web.Request, labelKey, labelValue string) error {
			ctx := request.Context()
			currentLabelValues := gjson.GetBytes(request.Body, fmt.Sprintf("labels.%s", labelKey)).Raw
			var path string
			var obj interface{}
			if len(currentLabelValues) != 0 {
				path = fmt.Sprintf("labels.%s.-1", labelKey)
				obj = labelValue
			} else {
				path = fmt.Sprintf("labels.%s", labelKey)
				obj = []string{labelValue}
			}

			var err error
			request.Body, err = sjson.SetBytes(request.Body, path, obj)
			if err != nil {
				return fmt.Errorf("could not add label with key %s and value %s: %s to request body during resource creation", labelKey, labelValue, err)
			}

			log.C(ctx).Infof("Successfully added label with key %s and value %s to request body during resource creation", labelKey, labelValue)

			return nil
		},
	}
}

// OIDCTenantCriteriaFilter filters the resources on GET, DELETE and PATCH during OAuth authentication based on the specified parameters
type OIDCTenantFilter struct {
	// ClientID are the client IDs of the OAuth clients, the tokens of which are issued for users with multi tenant access
	ClientID string
	// ClientIDTokenClaim is the token claim that contains the value of the client ID of the client from which the token was issued
	ClientIDTokenClaim string
	// LabelKey is the key of the label, the value of which will be used to filter the returned resources for this user
	LabelKey string
	// LabelTokenClaim is the token claim that contains the LabelKey value for the user for which the token was issued
	LabelTokenClaim string

	ProcessFunc func(request *web.Request, labelKey, labelValue string) error
	FilterName  string
	Methods     []string
}

// Name implements web.Named and returns the filter name
func (f *OIDCTenantFilter) Name() string {
	return f.FilterName
}

// Run implements web.Middleware and attempts to add additional label crieria to the context that would be used to filter the resources
func (f *OIDCTenantFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	logger := log.C(ctx)
	if len(f.ClientID) == 0 || f.ClientIDTokenClaim == "" || f.LabelKey == "" || f.LabelTokenClaim == "" {
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
	if f.ClientID != clientIDFromToken {
		logger.Infof("Token in user context was issued by %s and not by the tenant aware client %s. Proceeding with next filter...", clientIDFromToken, f.ClientID)
		return next.Handle(request)
	}

	delimiterClaimValue := gjson.GetBytes([]byte(userData), f.LabelTokenClaim).String()
	if len(delimiterClaimValue) == 0 {
		return nil, fmt.Errorf("invalid token: could not find delimiter %s in token claims", f.LabelTokenClaim)
	}

	if err := f.ProcessFunc(request, f.LabelKey, delimiterClaimValue); err != nil {
		return nil, err
	}

	return next.Handle(request)
}

// FilterMatchers implements web.Filter.FilterMatchers and specifies that the filter should run on GET, PATCH and DELETE for all resources
func (f *OIDCTenantFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Methods(f.Methods...),
			},
		},
	}
}
