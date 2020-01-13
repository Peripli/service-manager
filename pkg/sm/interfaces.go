package sm

import "context"

// Extendable provides a mechanism to extend further the Service Manager builder
type Extendable interface {
	Extend(context.Context, *ServiceManagerBuilder) error
}
