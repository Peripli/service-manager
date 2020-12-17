package storage

import "github.com/Peripli/service-manager/pkg/util"

type SubQuery int

const (
	QueryForAllLastOperationsPerResource SubQuery = iota
	QueryForOperationsWithResource
	QueryForTenantScopedServiceOfferings
	QueryForInstanceChildrenByLabel
)

// The sub-queries are dedicated to be used with ByExists/ByNotExists Criterion to allow additional querying/filtering
// combined with a main query generated from the criteria.
// As in sql's EXISTS/ NOT EXISTS, in order for the main query to work with the sub-query, the sub-queries will require a where clause that
// uses columns from the parent query for comparison.
//
// Example: Get all internal operations which aren't orphans (have corresponding resources) using ByExists criterion:
//
// queryForAllNonOrphanOperations := `
//    SELECT id
//    FROM service_instances
//	  WHERE operations.resource_id = service_instances.id`
//
// criteria := []query.Criterion{
// query.ByExists(queryForAllNonOrphanOperations),
// }
// allNonOrphanServiceOperations, _ := repository.List(ctx, types.OperationType, criteria...)

type SubQueryParams map[string]interface{}

var subQueries = map[SubQuery]string{
	QueryForAllLastOperationsPerResource: `
	SELECT id
    FROM operations op
    INNER JOIN (
        SELECT max(operations.paging_sequence) paging_sequence
        FROM operations
        GROUP BY resource_id, resource_type) LAST_OPERATIONS ON 
    op.paging_sequence = LAST_OPERATIONS.paging_sequence
    WHERE operations.id = op.id`,
	QueryForOperationsWithResource: `
    SELECT id
    FROM {{.RESOURCE_TABLE}}
    WHERE operations.resource_id = {{.RESOURCE_TABLE}}.id`,
	QueryForTenantScopedServiceOfferings: `
	SELECT id FROM broker_labels l
	WHERE l.broker_id = service_offerings.broker_id AND l.key = '{{.TENANT_KEY}}'`,
	QueryForInstanceChildrenByLabel: `
		SELECT 1 FROM service_instances i
        INNER JOIN service_instance_labels l ON i.id = l.service_instance_id
		WHERE  l.key IN ({{.PARENT_KEYS}}) AND l.val = '{{.PARENT_ID}}' AND i.id = service_instances.id`,
}

func GetSubQuery(query SubQuery) string {
	return subQueries[query]
}

func GetSubQueryWithParams(query SubQuery, params SubQueryParams) (string, error) {
	return util.Tsprintf(subQueries[query], params)
}
