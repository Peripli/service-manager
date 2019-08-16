package configuration

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

// Controller logging configuration controller
type Controller struct {
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
	if err := util.BytesToObject(r.Body, loggingConfig); err != nil {
		return nil, err
	}

	log.C(ctx).Infof("Attempting to set logging configuration with level: %s and format: %s", loggingConfig.Level, loggingConfig.Format)
	ctx = log.Configure(ctx, loggingConfig)
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
