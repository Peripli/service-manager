package postgres

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
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
	limit                        int
	criteria                     []query.Criterion
	hasLock                      bool
	returningFields              []string

	err error
}

func (qb *pgQuery) List(ctx context.Context, entity PostgresEntity) (*sqlx.Rows, error) {
	if qb.err != nil {
		return nil, qb.err
	}

	baseQuery := constructBaseQueryForLabelable(entity.LabelEntity(), entity.TableName())
	qb.sql.WriteString(baseQuery)

	if err := qb.finalizeSQL(entity); err != nil {
		return nil, err
	}

	return qb.db.QueryxContext(ctx, qb.sql.String(), qb.queryParams...)
}

func (qb *pgQuery) Delete(ctx context.Context, entity PostgresEntity) (*sqlx.Rows, error) {
	if qb.err != nil {
		return nil, qb.err
	}
	baseTableName := entity.TableName()
	qb.sql.WriteString(fmt.Sprintf("DELETE FROM %s", baseTableName))
	for len(qb.labelCriteria) > 0 {
		return nil, &util.UnsupportedQueryError{Message: "conditional delete is only supported for field queries"}
	}

	if err := qb.finalizeSQL(entity); err != nil {
		return nil, err
	}
	return qb.db.QueryxContext(ctx, qb.sql.String(), qb.queryParams...)
}

func (qb *pgQuery) Return(fields ...string) *pgQuery {
	qb.returningFields = append(qb.returningFields, fields...)

	return qb
}

func (qb *pgQuery) Limit(limit int) *pgQuery {
	if limit <= 0 {
		qb.err = fmt.Errorf("limit (%d) should be greater than 0", limit)
		return qb
	}

	qb.limit = limit

	return qb
}

func (qb *pgQuery) OrderBy(field string, orderType query.OrderType) *pgQuery {
	qb.orderByFields = append(qb.orderByFields, orderRule{
		field:     field,
		orderType: orderType,
	})

	return qb
}

func (qb *pgQuery) WithCriteria(criteria ...query.Criterion) *pgQuery {
	if len(criteria) == 0 {
		return qb
	}

	qb.criteria = append(qb.criteria, criteria...)
	labelCriteria, fieldCriteria, resultCriteria := splitCriteriaByType(criteria)
	qb.labelCriteria = append(qb.labelCriteria, labelCriteria...)
	qb.fieldCriteria = append(qb.fieldCriteria, fieldCriteria...)

	qb.processResultCriteria(resultCriteria)

	return qb
}

func (qb *pgQuery) WithLock() *pgQuery {
	if _, ok := qb.db.(*sqlx.Tx); ok {
		qb.hasLock = true
	}
	return qb
}

func (qb *pgQuery) finalizeSQL(entity PostgresEntity) error {
	entityTags := getDBTags(entity, nil)
	columns := columnsByTags(entityTags)
	if err := validateFieldQueryParams(columns, qb.criteria); err != nil {
		return err
	}
	if err := validateOrderFields(columns, qb.orderByFields...); err != nil {
		return err
	}
	if err := validateReturningFields(columns, qb.returningFields...); err != nil {
		return err
	}

	qb.labelCriteriaSQL(entity, qb.labelCriteria).
		fieldCriteriaSQL(entity, qb.fieldCriteria).
		orderBySQL().
		limitSQL().
		lockSQL(entity.TableName()).
		returningSQL().
		expandMultivariateOp()

	if qb.err != nil {
		return qb.err
	}

	sql := qb.sql.String()
	qb.sql.Reset()
	qb.sql.WriteString(qb.db.Rebind(sql))
	qb.sql.WriteString(";")
	return nil
}

func (qb *pgQuery) orderBySQL() *pgQuery {
	if len(qb.orderByFields) > 0 {
		sql := " ORDER BY"
		for _, orderRule := range qb.orderByFields {
			sql += fmt.Sprintf(" %s %s,", orderRule.field, qb.orderTypeToSQL(orderRule.orderType))
		}
		sql = sql[:len(sql)-1]
		qb.sql.WriteString(sql)
	}
	return qb
}

func (qb *pgQuery) limitSQL() *pgQuery {
	if qb.limit > 0 {
		qb.sql.WriteString(fmt.Sprintf(" LIMIT %d", qb.limit))
	}
	return qb
}

func (qb *pgQuery) returningSQL() *pgQuery {
	if len(qb.returningFields) == 1 && qb.returningFields[0] == "*" {
		qb.sql.WriteString(" RETURNING *")
	} else if len(qb.returningFields) > 0 {
		qb.sql.WriteString(" RETURNING " + strings.Join(qb.returningFields, ","))
	}
	return qb
}

func (qb *pgQuery) lockSQL(tableName string) *pgQuery {
	if qb.hasLock {
		// Lock the rows if we are in transaction so that update operations on those rows can rely on unchanged data
		// This allows us to handle concurrent updates on the same rows by executing them sequentially as
		// before updating we have to anyway select the rows and can therefore lock them
		qb.sql.WriteString(fmt.Sprintf(" FOR SHARE of %s", tableName))
	}
	return qb
}

func (qb *pgQuery) labelCriteriaSQL(entity PostgresEntity, criteria []query.Criterion) *pgQuery {
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
			qb.queryParams = append(qb.queryParams, option.LeftOp, rightOpQueryValue)
		}
		labelSubQuery += strings.Join(labelQueries, " OR ")
		labelSubQuery += "))"

		qb.sql.Replace("LEFT JOIN", "JOIN "+labelSubQuery)
	}
	return qb
}

func (qb *pgQuery) fieldCriteriaSQL(entity PostgresEntity, criteria []query.Criterion) *pgQuery {
	baseTableName := entity.TableName()
	dbTags := getDBTags(entity, nil)

	var fieldQueries []string

	if len(criteria) > 0 {
		qb.sql.WriteString(" WHERE ")
		for _, option := range criteria {
			var ttype reflect.Type
			if dbTags != nil {
				var err error
				ttype, err = findTagType(dbTags, option.LeftOp)
				if err != nil {
					qb.err = err
					return qb
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
			qb.queryParams = append(qb.queryParams, rightOpQueryValue)
		}
		qb.sql.WriteString(strings.Join(fieldQueries, " AND "))
	}
	return qb
}

func (qb *pgQuery) processResultCriteria(resultQuery []query.Criterion) *pgQuery {
	for _, c := range resultQuery {
		if c.Type != query.ResultQuery {
			qb.err = fmt.Errorf("result query is expected, but %s is provided", c.Type)
			return qb
		}
		switch c.LeftOp {
		case query.OrderBy:
			if len(c.RightOp) < 1 {
				qb.err = fmt.Errorf("order by clause expects field and order type, but has none")
				return qb
			}
			if len(c.RightOp) < 2 {
				qb.err = fmt.Errorf("order by clause (%s) expects order type, but has none", c.RightOp[0])
				return qb
			}
			qb.OrderBy(c.RightOp[0], query.OrderType(c.RightOp[1]))
		case query.Limit:
			limit, err := strconv.Atoi(c.RightOp[0])
			if err != nil {
				qb.err = err
				return qb
			}
			qb.Limit(limit)
		}
	}

	return qb
}

func (qb *pgQuery) expandMultivariateOp() *pgQuery {
	if hasMultiVariateOp(qb.criteria) {
		var err error
		// sqlx.In requires question marks(?) instead of positional arguments (the ones pgsql uses) in order to map the list argument to the IN operation
		var sql string
		if sql, qb.queryParams, err = sqlx.In(qb.sql.String(), qb.queryParams...); err != nil {
			qb.err = err
			return qb
		}
		qb.sql.Reset()
		qb.sql.WriteString(sql)
	}
	return qb
}

func (qb *pgQuery) orderTypeToSQL(orderType query.OrderType) string {
	switch orderType {
	case query.AscOrder:
		return "ASC"
	case query.DescOrder:
		return "DESC"
	default:
		qb.err = fmt.Errorf("unsupported order type: %s", string(orderType))
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
