package types

import (
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/pkg/util"
	"reflect"
)

//go:generate smgen api BlockedClient
// BlockedClient struct
type BlockedClient struct {
	Base
	ClientID       string   `json:"client_id"`
	SubaccountID   string   `json:"subaccount_id"`
	BlockedMethods []string `json:"blocked_methods,omitempty"`
}

func (e *BlockedClient) Equals(obj Object) bool {
	if !Equals(e, obj) {
		return false
	}

	blockedClient := obj.(*BlockedClient)
	if e.ClientID != blockedClient.ClientID ||
		e.SubaccountID != blockedClient.SubaccountID ||
		!reflect.DeepEqual(e.BlockedMethods, blockedClient.BlockedMethods) {
		return false
	}

	return true
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (e *BlockedClient) Validate() error {
	if util.HasRFC3986ReservedSymbols(e.ID) {
		return fmt.Errorf("%s contains invalid character(s)", e.ID)
	}
	if e.ClientID == "" {
		return errors.New("missing blocked client ID")
	}
	if e.SubaccountID == "" {
		return errors.New("missing blocked subaccount ID")
	}
	if err := e.Labels.Validate(); err != nil {
		return err
	}

	return nil
}
