package sm

import "context"

type Extendable interface {
	Extend(context.Context, *ServiceManagerBuilder) error
}
