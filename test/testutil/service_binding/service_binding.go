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
		CredentialsObject: nil,
		Name:              "test-service-binding",
		ServiceInstanceID: serviceInstanceID,
		SyslogDrainURL:    "",
		RouteServiceURL:   "",
		VolumeMounts:      nil,
		Endpoints:         nil,
		Context:           []byte(OSBContext),
		BindResource:      nil,
		Credentials:       credentials,
	}
}
