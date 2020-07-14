package storage

type SubQuery int

const (
	QueryForAllLastOperationsPerResource SubQuery = iota
	QueryForNonResourcelessOperations
)

// The aforementioned sub-queries are dedicated to be used with ByIDExists/ByIDNotExists Criterion to allow additional querying/filtering
// combined with a main query generated from the criteria.
// As in sql's EXIST/ NOT EXIST, in order for the main query to work with the sub-query, the sub-queries will require a where clause that
// compares the id from the parent query with and id retrieved from the sub-query.
//
// Example: Get all internal operations which aren't orphans (have corresponding resources) using ByIDExist criterion:
//
// queryForAllNonOrphanOperations := `
//    SELECT id
//    FROM service_instances.id
//	  WHERE operations.resource_id = service_instances.id`
//
// criteria := []query.Criterion{
// query.ByIDExist(queryForAllNonOrphanOperations),
// }
// allNonOrphanServiceOperations, _ := repository.List(ctx, types.OperationType, criteria...)

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
	QueryForNonResourcelessOperations: `
    SELECT id
    FROM {{.RESOURCE_TABLE}}
    WHERE operations.resource_id = {{.RESOURCE_TABLE}}.id`,
}

func GetSubQuery(query SubQuery) string {
	return subQueries[query]
}
