package cascadetypes

import (
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

type CascadeOperationCriterion interface {
	GetChildrenCriterion() map[types.ObjectType][]query.Criterion
}
