package configuration

import (
	"net/http"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/env"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

// Controller configuration controller
type Controller struct {
	Environment env.Environment
}

func (c *Controller) getConfiguration(r *web.Request) (*web.Response, error) {
	log.C(r.Context()).Debug("Obtaining application configuration...")

	return util.NewJSONResponse(http.StatusOK, c.Environment.AllSettings())
}

func (c *Controller) getLoggingConfiguration(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	logCfg := log.Configuration()
	log.C(ctx).Debugf("Obtaining log configuration with level: %s and format: %s", logCfg.Format, logCfg.Level)

	return util.NewJSONResponse(http.StatusOK, logCfg)
}

func (c *Controller) setLoggingConfiguration(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	loggingConfig := log.Configuration()
	if err := util.BytesToObject(r.Body, &loggingConfig); err != nil {
		return nil, err
	}

	log.C(ctx).Infof("Attempting to set logging configuration with level: %s and format: %s", loggingConfig.Level, loggingConfig.Format)

	if loggingConfig.Output != log.Configuration().Output {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "Changing logger output is not allowed",
			StatusCode:  http.StatusBadRequest,
		}
	}

	if _, err := log.Configure(ctx, &loggingConfig); err != nil {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: err.Error(),
			StatusCode:  http.StatusBadRequest,
		}
	}
	r.Request = r.WithContext(ctx)
	log.C(ctx).Infof("Successfully set logging configuration with level: %s and format: %s", loggingConfig.Level, loggingConfig.Format)

	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

// Routes provides endpoints for modifying and obtaining the logging configuration
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.ConfigURL,
			},
			Handler: c.getConfiguration,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.LoggingConfigURL,
			},
			Handler: c.getLoggingConfiguration,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPut,
				Path:   web.LoggingConfigURL,
			},
			Handler: c.setLoggingConfiguration,
		},
	}
}
