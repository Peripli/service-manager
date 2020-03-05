// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

const BrokerPlatformCredentialType ObjectType = web.BrokerPlatformCredentialsURL

type BrokerPlatformCredentials struct {
	BrokerPlatformCredentials []*BrokerPlatformCredential `json:"broker_platform_credentials"`
}

func (e *BrokerPlatformCredentials) Add(object Object) {
	e.BrokerPlatformCredentials = append(e.BrokerPlatformCredentials, object.(*BrokerPlatformCredential))
}

func (e *BrokerPlatformCredentials) ItemAt(index int) Object {
	return e.BrokerPlatformCredentials[index]
}

func (e *BrokerPlatformCredentials) Len() int {
	return len(e.BrokerPlatformCredentials)
}

func (e *BrokerPlatformCredential) GetType() ObjectType {
	return BrokerPlatformCredentialType
}

// MarshalJSON override json serialization for http response
func (e *BrokerPlatformCredential) MarshalJSON() ([]byte, error) {
	type E BrokerPlatformCredential
	toMarshal := struct {
		*E
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
		Labels    Labels  `json:"labels,omitempty"`
	}{
		E:      (*E)(e),
		Labels: e.Labels,
	}
	if !e.CreatedAt.IsZero() {
		str := util.ToRFCNanoFormat(e.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !e.UpdatedAt.IsZero() {
		str := util.ToRFCNanoFormat(e.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}
	hasNoLabels := true
	for key, values := range e.Labels {
		if key != "" && len(values) != 0 {
			hasNoLabels = false
			break
		}
	}
	if hasNoLabels {
		toMarshal.Labels = nil
	}
	return json.Marshal(toMarshal)
}
