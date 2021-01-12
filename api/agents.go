package api

import (
	"github.com/Peripli/service-manager/pkg/agents"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
)

type AgentsController struct {
	agentsConfig *agents.Settings
}

func NewAgentsController(agentsConfig *agents.Settings) *AgentsController {
	return &AgentsController{
		agentsConfig: agentsConfig,
	}
}
func (c *AgentsController) GetSupportedVersions(req *web.Request) (resp *web.Response, err error) {
	if c.agentsConfig != nil {
		return util.NewJSONResponse(http.StatusOK, c.agentsConfig.SupportedVersions)
	}
	return util.NewJSONResponse(http.StatusOK, map[string]string{})
}

func (c *AgentsController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.AgentsURL,
			},
			Handler: c.GetSupportedVersions,
		},
	}
}
