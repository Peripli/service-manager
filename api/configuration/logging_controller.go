package configuration

import (
	"net/http"
	"os"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

// LoggingConfig struct represents the configurable logger properties
type LoggingConfig struct {
	Level  string `json:"level,omitempty"`
	Format string `json:"format,omitempty"`
}

// Controller logging configuration controller
type Controller struct {
}

func (c *Controller) getLoggingConfiguration(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	logCfg := log.Configuration()
	log.C(ctx).Debugf("Obtaining log configuration: %+v", logCfg)

	return util.NewJSONResponse(http.StatusOK, &LoggingConfig{
		Level:  logCfg.Level,
		Format: logCfg.Format,
	})
}

func (c *Controller) setLoggingConfiguration(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	loggingConfig := &log.Settings{
		Output: os.Stdout,
	}
	body := &LoggingConfig{
		Format: "kibana",
	}
	if err := util.BytesToObject(r.Body, body); err != nil {
		return nil, err
	}

	loggingConfig.Format = body.Format
	loggingConfig.Level = body.Level

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
