// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/util"
)

const PlatformType ObjectType = "Platform"

type Platforms struct {
	Platforms []*Platform `json:"platforms"`
}

func (e *Platforms) Add(object Object) {
	e.Platforms = append(e.Platforms, object.(*Platform))
}

func (e *Platforms) ItemAt(index int) Object {
	return e.Platforms[index]
}

func (e *Platforms) Len() int {
	return len(e.Platforms)
}

func (e *Platform) EmptyList() ObjectList {
	return &Platforms{Platforms: make([]*Platform, 0)}
}

func (e *Platform) GetType() ObjectType {
	return PlatformType
}

// MarshalJSON override json serialization for http response
func (e *Platform) MarshalJSON() ([]byte, error) {
	type E Platform
	toMarshal := struct {
		*E
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
	}{
		E: (*E)(e),
	}
	if !e.CreatedAt.IsZero() {
		str := util.ToRFCFormat(e.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !e.UpdatedAt.IsZero() {
		str := util.ToRFCFormat(e.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}
	return json.Marshal(toMarshal)
}
