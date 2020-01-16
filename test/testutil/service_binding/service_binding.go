package service_binding

import (
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"

	. "github.com/onsi/ginkgo"
)

func Prepare(serviceInstanceID string, OSBContext string, credentials string) *types.ServiceBinding {
	bindingID, err := uuid.NewV4()
	if err != nil {
		Fail(fmt.Sprintf("failed to generate binding GUID: %s", err))
	}

	return &types.ServiceBinding{
		Base: types.Base{
			ID:        bindingID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Secured:           nil,
		Name:              "test-service-binding",
		ServiceInstanceID: serviceInstanceID,
		SyslogDrainURL:    "drain_url",
		RouteServiceURL:   "route_service_url",
		VolumeMounts:      []byte(`[]`),
		Endpoints:         []byte(`[]`),
		Context:           []byte(OSBContext),
		BindResource:      []byte(`{"app_guid": "app-guid"}`),
		Credentials:       credentials,
	}
}
