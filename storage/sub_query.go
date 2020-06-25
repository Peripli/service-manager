package storage

type SubQuery int

const (
	QueryForAllLastOperationsPerResource SubQuery = iota
)

var subQueries = map[SubQuery]string{
	QueryForAllLastOperationsPerResource: `
	SELECT id
    FROM operations op
    INNER JOIN (
        SELECT max(operations.paging_sequence) paging_sequence
        FROM operations
        GROUP BY resource_id ) LAST_OPERATIONS ON 
    op.paging_sequence = LAST_OPERATIONS.paging_sequence
    WHERE operations.id = op.id`,
}

func GetSubQuery(query SubQuery) string {
	return subQueries[query]
}
