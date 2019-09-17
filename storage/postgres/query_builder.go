package postgres

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/jmoiron/sqlx"
)

const mainTableAlias = "t"

type orderRule struct {
	field     string
	orderType query.OrderType
}

type logicalOperator string

const (
	AND logicalOperator = "AND"
	OR  logicalOperator = "OR"
)

// TODO: Handle errors on writeString
type queryStringBuilder struct {
	strings.Builder
}

func (qsb *queryStringBuilder) Replace(old, new string) {
	current := qsb.String()
	qsb.Reset()
	current = strings.Replace(current, old, new, 1)
	qsb.WriteString(current)
}

// whereClauseTree represents an sql where clause as tree structure with AND/OR on the nodes
type whereClauseTree struct {
	operator  logicalOperator
	criterion query.Criterion
	children  []*whereClauseTree
}

func newWhereClauseTree() *whereClauseTree {
	return &whereClauseTree{
		children: make([]*whereClauseTree, 0),
	}
}

func (t *whereClauseTree) addNode(operator logicalOperator) *whereClauseTree {
	node := &whereClauseTree{
		operator: operator,
		children: make([]*whereClauseTree, 0),
	}
	t.children = append(t.children, node)
	return node
}

func (t *whereClauseTree) addLeaf(criterion query.Criterion) {
	t.children = append(t.children, &whereClauseTree{
		criterion: criterion,
		children:  make([]*whereClauseTree, 0),
	})
}

func (t *whereClauseTree) isLeaf() bool {
	return len(t.children) == 0
}

func (t *whereClauseTree) compileSQL(dbTags []tagType) (string, []interface{}, error) {
	if t.isLeaf() {
		sql, queryParam, err := criterionSQL(t.criterion, dbTags)
		if err != nil {
			return "", nil, err
		}
		return sql, []interface{}{queryParam}, nil
	}
	queryParams := make([]interface{}, 0)
	childrenSQL := make([]string, 0)
	for _, child := range t.children {
		childSQL, childQueryParams, err := child.compileSQL(dbTags)
		if err != nil {
			return "", nil, err
		}
		childrenSQL = append(childrenSQL, childSQL)
		queryParams = append(queryParams, childQueryParams...)
	}
	sep := " " + string(t.operator) + " "
	sql := fmt.Sprintf("(%s)", strings.Join(childrenSQL, sep))
	return sql, queryParams, nil
}

type subQuery struct {
	tableName        string
	referenceColumns string
	dbTags           []tagType
	sql              queryStringBuilder
	queryParams      []interface{}

	fieldCriteria []query.Criterion
	orderByFields []orderRule
	limit         string

	whereClauseTreeCreator func([]query.Criterion) *whereClauseTree

	err error
}

type selectSubQuery struct {
	*subQuery
}

func newSelectSubQuery(tableName string, referenceColumns string, dbTags []tagType, creator func([]query.Criterion) *whereClauseTree) *selectSubQuery {
	return &selectSubQuery{
		&subQuery{
			tableName:              tableName,
			referenceColumns:       referenceColumns,
			dbTags:                 dbTags,
			whereClauseTreeCreator: creator,
		},
	}
}

func (ssq *selectSubQuery) orderBySQL() *selectSubQuery {
	if sql, err := orderBySQL(ssq.orderByFields); err != nil {
		ssq.err = err
		return ssq
	} else {
		ssq.sql.WriteString(sql)
	}
	return ssq
}

func (ssq *selectSubQuery) limitSQL() *selectSubQuery {
	if len(ssq.limit) > 0 {
		ssq.sql.WriteString(fmt.Sprintf(" LIMIT %s", ssq.limit))
	}
	return ssq
}

func (ssq *selectSubQuery) fieldCriteriaSQL() *selectSubQuery {
	if len(ssq.fieldCriteria) > 0 {
		tree := ssq.whereClauseTreeCreator(ssq.fieldCriteria)
		whereSQL, queryParams, err := tree.compileSQL(ssq.dbTags)
		if err != nil {
			ssq.err = err
			return ssq
		}
		ssq.queryParams = append(ssq.queryParams, queryParams...)
		ssq.sql.WriteString(" WHERE " + whereSQL)
	}
	return ssq
}

func (ssq *selectSubQuery) compileSQL() (string, error) {
	columns := columnsByTags(ssq.dbTags)
	if err := validateFieldQueryParams(columns, ssq.fieldCriteria); err != nil {
		return "", err
	}
	if err := validateOrderFields(columns, ssq.orderByFields...); err != nil {
		return "", err
	}
	baseQuery := fmt.Sprintf("(SELECT %s FROM %s", ssq.referenceColumns, ssq.tableName)
	ssq.sql.WriteString(baseQuery)

	ssq.fieldCriteriaSQL().
		orderBySQL().
		limitSQL()

	ssq.sql.WriteString(")")

	if ssq.err != nil {
		return "", ssq.err
	}
	return ssq.sql.String(), nil
}

func criterionSQL(criterion query.Criterion, dbTags []tagType) (string, interface{}, error) {
	var ttype reflect.Type
	if dbTags != nil {
		var err error
		ttype, err = findTagType(dbTags, criterion.LeftOp)
		if err != nil {
			return "", nil, err
		}
	}
	rightOpBindVar, rightOpQueryValue := buildRightOp(criterion)
	sqlOperation := translateOperationToSQLEquivalent(criterion.Operator)

	dbCast := determineCastByType(ttype)
	clause := fmt.Sprintf("%s%s %s %s", criterion.LeftOp, dbCast, sqlOperation, rightOpBindVar)
	if criterion.Operator.IsNullable() {
		clause = fmt.Sprintf("(%s OR %s IS NULL)", clause, criterion.LeftOp)
	}
	return clause, rightOpQueryValue, nil
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
func (qb *QueryBuilder) NewQuery(entity PostgresEntity) *pgQuery {
	fromSubquery := newSelectSubQuery(entity.TableName(), "*", getDBTags(entity, nil),
		func(criteria []query.Criterion) *whereClauseTree {
			tree := newWhereClauseTree()
			tree.operator = AND
			for _, criterion := range criteria {
				tree.addLeaf(criterion)
			}
			return tree
		})

	labelEntity := entity.LabelEntity()
	joinLabelsSubquery := &selectSubQuery{&subQuery{}}
	if labelEntity != nil {
		joinLabelsSubquery = newSelectSubQuery(labelEntity.LabelsTableName(), labelEntity.ReferenceColumn(), getDBTags(entity.LabelEntity(), nil),
			func(criteria []query.Criterion) *whereClauseTree {
				tree := newWhereClauseTree()
				tree.operator = OR
				for i := 0; i < len(criteria)-1; i += 2 {
					newNode := tree.addNode(AND)
					newNode.addLeaf(criteria[i])
					newNode.addLeaf(criteria[i+1])
				}
				return tree
			})
	}
	return &pgQuery{
		entity:             entity,
		db:                 qb.db,
		fromSubquery:       fromSubquery,
		joinLabelsSubquery: joinLabelsSubquery,
	}
}

// pgQuery is used to construct postgres queries. It should be constructed only via the query builder. It is not safe for concurrent use.
type pgQuery struct {
	db                 pgDB
	entity             PostgresEntity
	sql                queryStringBuilder
	fromSubquery       *selectSubQuery
	joinLabelsSubquery *selectSubQuery
	queryParams        []interface{}

	orderByFields   []orderRule
	hasLock         bool
	returningFields []string

	err error
}

func (pgq *pgQuery) List(ctx context.Context) (*sqlx.Rows, error) {
	if pgq.err != nil {
		return nil, pgq.err
	}
	table := pgq.entity.TableName()
	labelsEntity := pgq.entity.LabelEntity()

	baseQuery := fmt.Sprintf("SELECT %s.*", mainTableAlias)

	if labelsEntity != nil {
		baseQuery += pgq.selectColumnsSQL(labelsEntity)

		var err error
		if table, err = pgq.fromSubquery.compileSQL(); err != nil {
			return nil, err
		}
	}
	baseQuery += fmt.Sprintf(" FROM %s %s", table, mainTableAlias)

	if labelsEntity != nil {
		baseQuery += pgq.joinLabelsSQL(labelsEntity)
	}

	pgq.sql.WriteString(baseQuery)

	if err := pgq.finalizeSQL(ctx); err != nil {
		return nil, err
	}

	return pgq.db.QueryxContext(ctx, pgq.sql.String(), pgq.queryParams...)
}

func (pgq *pgQuery) Count(ctx context.Context) (int, error) {
	if pgq.err != nil {
		return 0, pgq.err
	}

	pgq.orderByFields = nil
	pgq.fromSubquery.orderByFields = nil

	table := pgq.entity.TableName()
	labelsEntity := pgq.entity.LabelEntity()

	baseQuery := fmt.Sprintf("SELECT COUNT(DISTINCT %[2]s.id) FROM %[1]s %[2]s", table, mainTableAlias)

	if labelsEntity != nil {
		var err error
		if table, err = pgq.fromSubquery.compileSQL(); err != nil {
			return 0, err
		}
		baseQuery = fmt.Sprintf("SELECT COUNT(DISTINCT %[2]s.id) FROM %[1]s %[2]s", table, mainTableAlias)
		baseQuery += pgq.joinLabelsSQL(labelsEntity)
	}

	pgq.sql.WriteString(baseQuery)

	if err := pgq.finalizeSQL(ctx); err != nil {
		return 0, err
	}

	var count int
	err := pgq.db.GetContext(ctx, &count, pgq.sql.String(), pgq.queryParams...)
	return count, err
}

func (pgq *pgQuery) Delete(ctx context.Context) (*sqlx.Rows, error) {
	if pgq.err != nil {
		return nil, pgq.err
	}

	table := pgq.entity.TableName()
	labelsEntity := pgq.entity.LabelEntity()

	baseQuery := fmt.Sprintf("DELETE FROM %[1]s USING %[1]s %[2]s", table, mainTableAlias)

	primaryKeyColumn := "id"
	if labelsEntity != nil {
		var err error
		if table, err = pgq.fromSubquery.compileSQL(); err != nil {
			return nil, err
		}
		baseQuery = fmt.Sprintf("DELETE FROM %s USING %s %s", pgq.entity.TableName(), table, mainTableAlias)
		baseQuery += pgq.joinLabelsSQL(labelsEntity)
	}

	baseQuery += fmt.Sprintf(` WHERE %[1]s.%[2]s = %[3]s.%[2]s`, mainTableAlias, primaryKeyColumn, pgq.entity.TableName())

	pgq.sql.WriteString(baseQuery)

	if err := pgq.finalizeSQL(ctx); err != nil {
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

	labelCriteria, fieldCriteria, resultCriteria := splitCriteriaByType(criteria)
	for _, criterion := range labelCriteria {
		pgq.joinLabelsSubquery.fieldCriteria = append(pgq.joinLabelsSubquery.fieldCriteria,
			query.ByField(query.EqualsOperator, "key", criterion.LeftOp),
			query.ByField(criterion.Operator, "val", criterion.RightOp...))
	}
	pgq.fromSubquery.fieldCriteria = append(pgq.fromSubquery.fieldCriteria, fieldCriteria...)

	pgq.processResultCriteria(resultCriteria)

	return pgq
}

func (pgq *pgQuery) WithLock() *pgQuery {
	if _, ok := pgq.db.(*sqlx.Tx); ok {
		pgq.hasLock = true
	}
	return pgq
}

func (pgq *pgQuery) selectColumnsSQL(labelsEntity PostgresLabel) string {
	labelsTableName := labelsEntity.LabelsTableName()
	baseQuery := `, `
	for _, dbTag := range getDBTags(labelsEntity, nil) {
		baseQuery += fmt.Sprintf(`%[1]s.%[2]s "%[1]s.%[2]s", `, labelsTableName, dbTag.Tag)
	}
	return baseQuery[:len(baseQuery)-2] //remove last comma
}

func (pgq *pgQuery) joinLabelsSQL(labelsEntity PostgresLabel) string {
	return fmt.Sprintf(` LEFT JOIN %[2]s ON %[1]s.%[3]s = %[2]s.%[4]s`,
		mainTableAlias,
		labelsEntity.LabelsTableName(),
		labelsEntity.LabelsPrimaryColumn(),
		labelsEntity.ReferenceColumn())
}

func (pgq *pgQuery) finalizeSQL(ctx context.Context) error {
	entityTags := getDBTags(pgq.entity, nil)
	columns := columnsByTags(entityTags)
	if err := validateReturningFields(columns, pgq.returningFields...); err != nil {
		return err
	}

	pgq.labelCriteriaSQL().
		lockSQL(pgq.entity.TableName()).
		orderBySQL().
		returningSQL().
		mergeQueryParams()

	if pgq.err != nil {
		return pgq.err
	}

	sql := pgq.sql.String()
	pgq.sql.Reset()
	pgq.sql.WriteString(pgq.db.Rebind(sql))
	pgq.sql.WriteString(";")

	log.C(ctx).Debugf("Executing postgres query: %s", pgq.sql.String())
	return nil
}

func (pgq *pgQuery) orderBySQL() *pgQuery {
	if sql, err := orderBySQL(pgq.orderByFields); err != nil {
		pgq.err = err
		return pgq
	} else {
		pgq.sql.WriteString(sql)
	}
	return pgq
}

func (pgq *pgQuery) returningSQL() *pgQuery {
	for i := range pgq.returningFields {
		if !strings.HasPrefix(pgq.returningFields[i], fmt.Sprintf("%s.", mainTableAlias)) {
			pgq.returningFields[i] = fmt.Sprintf("%s.%s", mainTableAlias, pgq.returningFields[i])
		}
	}

	if len(pgq.returningFields) == 1 {
		pgq.sql.WriteString(fmt.Sprintf(" RETURNING " + pgq.returningFields[0]))
	} else if len(pgq.returningFields) > 0 {
		pgq.sql.WriteString(" RETURNING " + strings.Join(pgq.returningFields, ", "))
	}
	return pgq
}

func (pgq *pgQuery) lockSQL(tableName string) *pgQuery {
	if pgq.hasLock {
		// Lock the rows if we are in transaction so that update operations on those rows can rely on unchanged data
		// This allows us to handle concurrent updates on the same rows by executing them sequentially as
		// before updating we have to anyway select the rows and can therefore lock them
		pgq.sql.WriteString(fmt.Sprintf(" FOR SHARE of %s", mainTableAlias))
	}
	return pgq
}

func (pgq *pgQuery) labelCriteriaSQL() *pgQuery {
	if len(pgq.joinLabelsSubquery.fieldCriteria) == 0 {
		return pgq
	}
	labelEntity := pgq.entity.LabelEntity()
	subquerySQL, err := pgq.joinLabelsSubquery.compileSQL()
	if err != nil {
		pgq.err = err
		return pgq
	}
	labelSubQuery := fmt.Sprintf("(SELECT * FROM %[1]s WHERE %[2]s IN %s)", labelEntity.LabelsTableName(), labelEntity.ReferenceColumn(), subquerySQL)
	pgq.sql.Replace("LEFT JOIN", "JOIN "+labelSubQuery)
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
			pgq.fromSubquery.orderByFields = append(pgq.fromSubquery.orderByFields, orderRule{
				field:     c.RightOp[0],
				orderType: query.OrderType(c.RightOp[1]),
			})
		case query.Limit:
			if pgq.fromSubquery.limit != "" {
				pgq.err = fmt.Errorf("zero/one limit expected but multiple provided")
				return pgq
			}
			pgq.fromSubquery.limit = c.RightOp[0]
		}
	}
	return pgq
}

func (pgq *pgQuery) mergeQueryParams() *pgQuery {
	if hasMultiVariateOp(append(pgq.fromSubquery.fieldCriteria, pgq.joinLabelsSubquery.fieldCriteria...)) {
		var err error
		// sqlx.In requires question marks(?) instead of positional arguments (the ones pgsql uses) in order to map the list argument to the IN operation
		var sql string
		if sql, pgq.queryParams, err = sqlx.In(pgq.sql.String(), append(pgq.fromSubquery.queryParams, pgq.joinLabelsSubquery.queryParams...)...); err != nil {
			pgq.err = err
			return pgq
		}
		pgq.sql.Reset()
		pgq.sql.WriteString(sql)
	} else {
		pgq.queryParams = append(pgq.fromSubquery.queryParams, pgq.joinLabelsSubquery.queryParams...)
	}
	return pgq
}

func orderBySQL(rules []orderRule) (string, error) {
	sql := ""
	if len(rules) > 0 {
		sql += " ORDER BY"
		for _, orderRule := range rules {
			orderType, err := orderTypeToSQL(orderRule.orderType)
			if err != nil {
				return "", err
			}
			sql += fmt.Sprintf(" %s %s,", orderRule.field, orderType)
		}
		sql = sql[:len(sql)-1]
	}
	return sql, nil
}

func orderTypeToSQL(orderType query.OrderType) (string, error) {
	var err error
	switch orderType {
	case query.AscOrder:
		return "ASC", nil
	case query.DescOrder:
		return "DESC", nil
	default:
		err = fmt.Errorf("unsupported order type: %s", string(orderType))
	}
	return "", err
}

func validateOrderFields(columns map[string]bool, orderRules ...orderRule) error {
	fields := make([]string, 0, len(orderRules))
	for _, or := range orderRules {
		fields = append(fields, or.field)
	}
	return validateFields(columns, "unsupported entity field for order by: %s", fields...)
}

func validateReturningFields(columns map[string]bool, returningFields ...string) error {
	fieldsToValidate := make([]string, 0, len(returningFields))
	for _, returnedField := range returningFields {
		if strings.Contains(returnedField, "*") {
			continue
		}
		fieldsToValidate = append(fieldsToValidate, returnedField)
	}
	return validateFields(columns, "unsupported entity field for return type: %s", fieldsToValidate...)
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
