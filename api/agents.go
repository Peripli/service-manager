package api

import (
	"encoding/json"
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
	if c.agentsConfig != nil && len(c.agentsConfig.Versions) > 0 {
		var supportedVersions = make(map[string]interface{})
		if err := json.Unmarshal([]byte(c.agentsConfig.Versions), &supportedVersions); err != nil {
			panic(err)
		}
		return util.NewJSONResponse(http.StatusOK, supportedVersions)
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
