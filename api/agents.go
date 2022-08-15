package api

import (
	"encoding/json"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/agents"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
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
		var supportedVersions = make(map[string][]string)
		if err := json.Unmarshal([]byte(c.agentsConfig.Versions), &supportedVersions); err != nil {
			return nil, &util.HTTPError{
				ErrorType:   http.StatusText(http.StatusInternalServerError),
				Description: "failed to retrieve agents supported versions",
				StatusCode:  http.StatusInternalServerError,
			}

		}
		return util.NewJSONResponse(http.StatusOK, supportedVersions)
	}
	return util.NewJSONResponse(http.StatusOK, map[string][]string{})
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
