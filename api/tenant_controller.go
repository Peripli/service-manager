package api

import (
	"fmt"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
	"net/http"
)

type TenantController struct {
	repository storage.Repository
}

func NewTenantController(repository storage.Repository) *TenantController {
	return &TenantController{repository: repository}
}

func (c *TenantController) GetOperation(req *web.Request) (resp *web.Response, err error) {
	return GetResourceOperation(req, c.repository, types.TenantType)
}

func (c *TenantController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}%s/{%s}", web.TenantURL, web.PathParamResourceID, web.ResourceOperationsURL, web.PathParamID),
			},
			Handler: c.GetOperation,
		},
	}
}
