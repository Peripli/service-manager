package postgres

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/jmoiron/sqlx"
)

type orderRule struct {
	field     string
	orderType query.OrderType
}

type queryStringBuilder struct {
	strings.Builder
}

func (qsb *queryStringBuilder) Replace(old, new string) {
	current := qsb.String()
	qsb.Reset()
	current = strings.Replace(current, old, new, 1)
	qsb.WriteString(current)
}

// QueryBuilder is used to construct new queries. It is safe for concurrent usage
type QueryBuilder struct {
	db pgDB
}

// NewQueryBuilder constructs new query builder for the current db
func NewQueryBuilder(db pgDB) *QueryBuilder {
	return &QueryBuilder{
		db: db,
	}
}

// NewQuery constructs new queries for the current query builder db
func (qb *QueryBuilder) NewQuery() *pgQuery {
	return &pgQuery{
		db: qb.db,
	}
}

// pgQuery is used to construct postgres queries. It should be constructed only via the query builder. It is not safe for concurrent use.
type pgQuery struct {
	db          pgDB
	sql         queryStringBuilder
	queryParams []interface{}

	labelCriteria, fieldCriteria []query.Criterion
	orderByFields                []orderRule
	limit                        string
	criteria                     []query.Criterion
	hasLock                      bool
	returningFields              []string

	err error
}

func (pgq *pgQuery) List(ctx context.Context, entity PostgresEntity) (*sqlx.Rows, error) {
	if pgq.err != nil {
		return nil, pgq.err
	}

	baseQuery := constructBaseQueryForLabelable(entity.LabelEntity(), entity.TableName())
	pgq.sql.WriteString(baseQuery)

	if err := pgq.finalizeSQL(entity); err != nil {
		return nil, err
	}

	return pgq.db.QueryxContext(ctx, pgq.sql.String(), pgq.queryParams...)
}

func (pgq *pgQuery) Delete(ctx context.Context, entity PostgresEntity) (*sqlx.Rows, error) {
	if pgq.err != nil {
		return nil, pgq.err
	}
	baseTableName := entity.TableName()
	pgq.sql.WriteString(fmt.Sprintf("DELETE FROM %s", baseTableName))
	for len(pgq.labelCriteria) > 0 {
		return nil, &util.UnsupportedQueryError{Message: "conditional delete is only supported for field queries"}
	}

	if err := pgq.finalizeSQL(entity); err != nil {
		return nil, err
	}
	return pgq.db.QueryxContext(ctx, pgq.sql.String(), pgq.queryParams...)
}

func (pgq *pgQuery) Return(fields ...string) *pgQuery {
	pgq.returningFields = append(pgq.returningFields, fields...)

	return pgq
}

func (pgq *pgQuery) WithCriteria(criteria ...query.Criterion) *pgQuery {
	if len(criteria) == 0 {
		return pgq
	}

	if err := validateCriteria(criteria...); err != nil {
		pgq.err = err
		return pgq
	}

	pgq.criteria = append(pgq.criteria, criteria...)
	labelCriteria, fieldCriteria, resultCriteria := splitCriteriaByType(criteria)
	pgq.labelCriteria = append(pgq.labelCriteria, labelCriteria...)
	pgq.fieldCriteria = append(pgq.fieldCriteria, fieldCriteria...)

	pgq.processResultCriteria(resultCriteria)

	return pgq
}

func (pgq *pgQuery) WithLock() *pgQuery {
	if _, ok := pgq.db.(*sqlx.Tx); ok {
		pgq.hasLock = true
	}
	return pgq
}

func (pgq *pgQuery) finalizeSQL(entity PostgresEntity) error {
	entityTags := getDBTags(entity, nil)
	columns := columnsByTags(entityTags)
	if err := validateFieldQueryParams(columns, pgq.criteria); err != nil {
		return err
	}
	if err := validateOrderFields(columns, pgq.orderByFields...); err != nil {
		return err
	}
	if err := validateReturningFields(columns, pgq.returningFields...); err != nil {
		return err
	}

	pgq.labelCriteriaSQL(entity, pgq.labelCriteria).
		fieldCriteriaSQL(entity, pgq.fieldCriteria).
		orderBySQL().
		limitSQL().
		lockSQL(entity.TableName()).
		returningSQL().
		expandMultivariateOp()

	if pgq.err != nil {
		return pgq.err
	}

	sql := pgq.sql.String()
	pgq.sql.Reset()
	pgq.sql.WriteString(pgq.db.Rebind(sql))
	pgq.sql.WriteString(";")
	return nil
}

func (pgq *pgQuery) orderBySQL() *pgQuery {
	if len(pgq.orderByFields) > 0 {
		sql := " ORDER BY"
		for _, orderRule := range pgq.orderByFields {
			sql += fmt.Sprintf(" %s %s,", orderRule.field, pgq.orderTypeToSQL(orderRule.orderType))
		}
		sql = sql[:len(sql)-1]
		pgq.sql.WriteString(sql)
	}
	return pgq
}

func (pgq *pgQuery) limitSQL() *pgQuery {
	if len(pgq.limit) > 0 {
		pgq.sql.WriteString(fmt.Sprintf(" LIMIT %s", pgq.limit))
	}
	return pgq
}

func (pgq *pgQuery) returningSQL() *pgQuery {
	if len(pgq.returningFields) == 1 && pgq.returningFields[0] == "*" {
		pgq.sql.WriteString(" RETURNING *")
	} else if len(pgq.returningFields) > 0 {
		pgq.sql.WriteString(" RETURNING " + strings.Join(pgq.returningFields, ","))
	}
	return pgq
}

func (pgq *pgQuery) lockSQL(tableName string) *pgQuery {
	if pgq.hasLock {
		// Lock the rows if we are in transaction so that update operations on those rows can rely on unchanged data
		// This allows us to handle concurrent updates on the same rows by executing them sequentially as
		// before updating we have to anyway select the rows and can therefore lock them
		pgq.sql.WriteString(fmt.Sprintf(" FOR SHARE of %s", tableName))
	}
	return pgq
}

func (pgq *pgQuery) labelCriteriaSQL(entity PostgresEntity, criteria []query.Criterion) *pgQuery {
	var labelQueries []string

	labelEntity := entity.LabelEntity()
	if len(criteria) > 0 {
		labelTableName := labelEntity.LabelsTableName()
		referenceColumnName := labelEntity.ReferenceColumn()
		labelSubQuery := fmt.Sprintf("(SELECT * FROM %[1]s WHERE %[2]s IN (SELECT %[2]s FROM %[1]s WHERE ", labelTableName, referenceColumnName)
		for _, option := range criteria {
			rightOpBindVar, rightOpQueryValue := buildRightOp(option)
			sqlOperation := translateOperationToSQLEquivalent(option.Operator)
			labelQueries = append(labelQueries, fmt.Sprintf("(%[1]s.key = ? AND %[1]s.val %[2]s %s)", labelTableName, sqlOperation, rightOpBindVar))
			pgq.queryParams = append(pgq.queryParams, option.LeftOp, rightOpQueryValue)
		}
		labelSubQuery += strings.Join(labelQueries, " OR ")
		labelSubQuery += "))"

		pgq.sql.Replace("LEFT JOIN", "JOIN "+labelSubQuery)
	}
	return pgq
}

func (pgq *pgQuery) fieldCriteriaSQL(entity PostgresEntity, criteria []query.Criterion) *pgQuery {
	baseTableName := entity.TableName()
	dbTags := getDBTags(entity, nil)

	var fieldQueries []string

	if len(criteria) > 0 {
		pgq.sql.WriteString(" WHERE ")
		for _, option := range criteria {
			var ttype reflect.Type
			if dbTags != nil {
				var err error
				ttype, err = findTagType(dbTags, option.LeftOp)
				if err != nil {
					pgq.err = err
					return pgq
				}
			}
			rightOpBindVar, rightOpQueryValue := buildRightOp(option)
			sqlOperation := translateOperationToSQLEquivalent(option.Operator)

			dbCast := determineCastByType(ttype)
			clause := fmt.Sprintf("%s.%s%s %s %s", baseTableName, option.LeftOp, dbCast, sqlOperation, rightOpBindVar)
			if option.Operator.IsNullable() {
				clause = fmt.Sprintf("(%s OR %s.%s IS NULL)", clause, baseTableName, option.LeftOp)
			}
			fieldQueries = append(fieldQueries, clause)
			pgq.queryParams = append(pgq.queryParams, rightOpQueryValue)
		}
		pgq.sql.WriteString(strings.Join(fieldQueries, " AND "))
	}
	return pgq
}

func (pgq *pgQuery) processResultCriteria(resultQuery []query.Criterion) *pgQuery {
	for _, c := range resultQuery {
		if c.Type != query.ResultQuery {
			pgq.err = fmt.Errorf("result query is expected, but %s is provided", c.Type)
			return pgq
		}
		switch c.LeftOp {
		case query.OrderBy:
			pgq.orderByFields = append(pgq.orderByFields, orderRule{
				field:     c.RightOp[0],
				orderType: query.OrderType(c.RightOp[1]),
			})
		case query.Limit:
			pgq.limit = c.RightOp[0]
		}
	}

	return pgq
}

func (pgq *pgQuery) expandMultivariateOp() *pgQuery {
	if hasMultiVariateOp(pgq.criteria) {
		var err error
		// sqlx.In requires question marks(?) instead of positional arguments (the ones pgsql uses) in order to map the list argument to the IN operation
		var sql string
		if sql, pgq.queryParams, err = sqlx.In(pgq.sql.String(), pgq.queryParams...); err != nil {
			pgq.err = err
			return pgq
		}
		pgq.sql.Reset()
		pgq.sql.WriteString(sql)
	}
	return pgq
}

func (pgq *pgQuery) orderTypeToSQL(orderType query.OrderType) string {
	switch orderType {
	case query.AscOrder:
		return "ASC"
	case query.DescOrder:
		return "DESC"
	default:
		pgq.err = fmt.Errorf("unsupported order type: %s", string(orderType))
	}
	return ""
}

func validateOrderFields(columns map[string]bool, orderRules ...orderRule) error {
	fields := make([]string, 0, len(orderRules))
	for _, or := range orderRules {
		fields = append(fields, or.field)
	}
	return validateFields(columns, "unsupported entity field for order by: %s", fields...)
}

func validateReturningFields(columns map[string]bool, returningFields ...string) error {
	if len(returningFields) > 0 {
		if returningFields[0] == "*" {
			return nil
		}
		return validateFields(columns, "unsupported entity field for return type: %s", returningFields...)
	}
	return nil
}

func validateFields(columns map[string]bool, errorTemplate string, fields ...string) error {
	for _, field := range fields {
		if !columns[field] {
			return &util.UnsupportedQueryError{Message: fmt.Sprintf(errorTemplate, field)}
		}
	}
	return nil
}

func validateCriteria(criteria ...query.Criterion) error {
	for _, c := range criteria {
		if err := c.Validate(); err != nil {
			return err
		}
	}
	return nil
}
