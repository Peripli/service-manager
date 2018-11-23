package service_offering

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

const reqServiceOfferingID = "service_offering_id"

type Controller struct {
	ServiceOfferingStorage storage.ServiceOffering
}

func (c *Controller) getServiceOffering(r *web.Request) (*web.Response, error) {
	serviceOfferingID := r.PathParams[reqServiceOfferingID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting service offering with id %s", serviceOfferingID)

	serviceOffering, err := c.ServiceOfferingStorage.Get(ctx, serviceOfferingID)
	if err = util.HandleStorageError(err, "service_offering", serviceOfferingID); err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusOK, serviceOffering)
}

func (c *Controller) listServiceOfferings(r *web.Request) (*web.Response, error) {
	var serviceOfferings []*types.ServiceOffering
	var err error
	ctx := r.Context()
	log.C(ctx).Debug("Listing service offerings")

	query := r.URL.Query()
	catalogName := query.Get("catalog_name")
	if catalogName != "" {
		log.C(ctx).Debugf("Filtering list by catalog_name=%s", catalogName)
		serviceOfferings, err = c.ServiceOfferingStorage.ListByCatalogName(ctx, catalogName)
	} else {
		serviceOfferings, err = c.ServiceOfferingStorage.List(ctx)
	}
	if err != nil {
		return nil, err
	}

	return util.NewJSONResponse(http.StatusOK, struct {
		ServiceOfferings []*types.ServiceOffering `json:"service_offerings"`
	}{
		ServiceOfferings: serviceOfferings,
	})
}
