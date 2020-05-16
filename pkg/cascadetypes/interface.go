package cascadetypes

import (
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

type CascadeOperationCriteria interface {
	GetChildrenCriteria() map[types.ObjectType][]query.Criterion
}
