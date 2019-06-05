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

type QueryBuilder struct {
	db pgDB

	sql         string
	queryParams []interface{}

	labelCriteria, fieldCriteria []query.Criterion
	orderByFields                []orderRule
	limit                        int
	criteria                     []query.Criterion
	hasLock                      bool
	returningFields              []string

	err error
}

func NewQueryBuilder(db pgDB) *QueryBuilder {
	return &QueryBuilder{
		db: db,
	}
}

func (qb *QueryBuilder) List(ctx context.Context, entity PostgresEntity) (*sqlx.Rows, error) {
	if qb.err != nil {
		return nil, qb.err
	}

	baseTableName := entity.TableName()
	var baseQuery string
	label := entity.LabelEntity()
	if label == nil {
		baseQuery = constructBaseQueryForEntity(baseTableName)
	} else {
		baseQuery = constructBaseQueryForLabelable(label, baseTableName)
	}
	qb.sql = baseQuery

	if err := qb.finalizeSQL(entity); err != nil {
		return nil, err
	}

	return qb.db.QueryxContext(ctx, qb.sql, qb.queryParams...)
}

func (qb *QueryBuilder) Delete(ctx context.Context, entity PostgresEntity) (*sqlx.Rows, error) {
	if qb.err != nil {
		return nil, qb.err
	}
	baseTableName := entity.TableName()
	qb.sql = fmt.Sprintf("DELETE FROM %s", baseTableName)
	for len(qb.labelCriteria) > 0 {
		return nil, &util.UnsupportedQueryError{Message: "conditional delete is only supported for field queries"}
	}

	if err := qb.finalizeSQL(entity); err != nil {
		return nil, err
	}
	return qb.db.QueryxContext(ctx, qb.sql, qb.queryParams...)
}

func (qb *QueryBuilder) finalizeSQL(entity PostgresEntity) error {
	if err := validateFieldQueryParams(getDBTags(entity, nil), qb.criteria); err != nil {
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

	qb.sql = qb.db.Rebind(qb.sql)
	qb.sql += ";"
	return nil
}

func (qb *QueryBuilder) Return(fields ...string) *QueryBuilder {
	qb.returningFields = append(qb.returningFields, fields...)
	return qb
}

func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	if qb.err != nil {
		return qb
	}

	if limit <= 0 {
		qb.err = fmt.Errorf("limit (%d) should be greater than 0", limit)
		return qb
	}

	qb.limit = limit

	return qb
}

func (qb *QueryBuilder) OrderBy(field string, orderType query.OrderType) *QueryBuilder {
	if qb.err != nil {
		return qb
	}

	qb.orderByFields = append(qb.orderByFields, orderRule{
		field:     field,
		orderType: orderType,
	})

	return qb
}

func (qb *QueryBuilder) WithCriteria(criteria ...query.Criterion) *QueryBuilder {
	if qb.err != nil {
		return qb
	}

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

func (qb *QueryBuilder) WithLock() *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	if _, ok := qb.db.(*sqlx.Tx); ok {
		qb.hasLock = true
	}
	return qb
}

func (qb *QueryBuilder) processResultCriteria(resultQuery []query.Criterion) *QueryBuilder {
	for _, c := range resultQuery {
		if c.Type != query.ResultQuery {
			qb.err = fmt.Errorf("result query is expected, but %s is provided", c.Type)
			return qb
		}
		switch c.LeftOp {
		case query.OrderBy:
			if len(c.RightOp) < 2 {
				qb.err = fmt.Errorf("order by clause expects ordering type, but has none")
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

func (qb *QueryBuilder) orderBySQL() *QueryBuilder {
	if len(qb.orderByFields) > 0 {
		sql := " ORDER BY"
		for _, orderRule := range qb.orderByFields {
			sql += fmt.Sprintf(" %s %s,", orderRule.field, orderTypeToSQL(orderRule.orderType))
		}
		sql = sql[:len(sql)-1]
		qb.sql += sql
	}
	return qb
}

func (qb *QueryBuilder) limitSQL() *QueryBuilder {
	if qb.limit > 0 {
		qb.sql += fmt.Sprintf(" LIMIT %d", qb.limit)
	}
	return qb
}

func (qb *QueryBuilder) returningSQL() *QueryBuilder {
	if len(qb.returningFields) == 1 && qb.returningFields[0] == "*" {
		qb.sql += " RETURNING *"
	} else if len(qb.returningFields) > 0 {
		qb.sql += " RETURNING " + strings.Join(qb.returningFields, ",")
	}
	return qb
}

func (qb *QueryBuilder) lockSQL(tableName string) *QueryBuilder {
	if qb.hasLock {
		// Lock the rows if we are in transaction so that update operations on those rows can rely on unchanged data
		// This allows us to handle concurrent updates on the same rows by executing them sequentially as
		// before updating we have to anyway select the rows and can therefore lock them
		qb.sql += fmt.Sprintf(" FOR SHARE of %s", tableName)
	}
	return qb
}

func (qb *QueryBuilder) labelCriteriaSQL(entity PostgresEntity, criteria []query.Criterion) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
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

		qb.sql = strings.Replace(qb.sql, "LEFT JOIN", "JOIN "+labelSubQuery, 1)
	}
	return qb
}

func (qb *QueryBuilder) fieldCriteriaSQL(entity PostgresEntity, criteria []query.Criterion) *QueryBuilder {
	if qb.err != nil {
		return qb
	}
	baseTableName := entity.TableName()
	dbTags := getDBTags(entity, nil)

	var fieldQueries []string

	if len(criteria) > 0 {
		qb.sql += " WHERE "
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
		qb.sql += strings.Join(fieldQueries, " AND ")
	}
	return qb
}

func (qb *QueryBuilder) expandMultivariateOp() *QueryBuilder {
	if hasMultiVariateOp(qb.criteria) {
		var err error
		// sqlx.In requires question marks(?) instead of positional arguments (the ones pgsql uses) in order to map the list argument to the IN operation
		if qb.sql, qb.queryParams, err = sqlx.In(qb.sql, qb.queryParams...); err != nil {
			qb.err = err
		}
	}
	return qb
}

func orderTypeToSQL(orderType query.OrderType) string {
	switch orderType {
	case query.AscOrder:
		return "ASC"
	case query.DescOrder:
		return "DESC"
	}
	return ""
}
