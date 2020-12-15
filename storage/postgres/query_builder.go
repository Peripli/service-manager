package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/storage"
	"regexp"
	"strings"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/jmoiron/sqlx"
)

const PrimaryKeyColumn = "id"

const CountQueryTemplate = `
SELECT COUNT(DISTINCT {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}})
FROM {{.ENTITY_TABLE}}
	{{if .hasLabelCriteria}}
	{{.JOIN}} {{.LABELS_TABLE}}
		ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
	{{end}}
{{.WHERE}}
{{.FOR_UPDATE_OF}}
{{.LIMIT}};`

const CountLabelValuesQueryTemplate = `
SELECT COUNT(DISTINCT {{.LABELS_TABLE}}.{{.PRIMARY_KEY}})
FROM {{.ENTITY_TABLE}}
	INNER JOIN {{.LABELS_TABLE}}
		ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
{{.WHERE}}
{{.FOR_UPDATE_OF}}
{{.LIMIT}};`

const SelectQueryTemplate = `
{{if or .hasFieldCriteria .hasLabelCriteria}}
WITH matching_resources as (SELECT DISTINCT {{.ENTITY_TABLE}}.paging_sequence
							FROM {{.ENTITY_TABLE}}
							{{if .hasLabelCriteria}}
							{{.JOIN}} {{.LABELS_TABLE}} 
								ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
							{{end}}
							{{.WHERE}}
							{{.ORDER_BY_SEQUENCE}}
							{{.LIMIT}})
{{end}}
SELECT 
{{.ENTITY_TABLE}}.*,
{{.LABELS_TABLE}}.id         "{{.LABELS_TABLE}}.id",
{{.LABELS_TABLE}}.key        "{{.LABELS_TABLE}}.key",
{{.LABELS_TABLE}}.val        "{{.LABELS_TABLE}}.val",
{{.LABELS_TABLE}}.created_at "{{.LABELS_TABLE}}.created_at",
{{.LABELS_TABLE}}.updated_at "{{.LABELS_TABLE}}.updated_at",
{{.LABELS_TABLE}}.{{.REF_COLUMN}} "{{.LABELS_TABLE}}.{{.REF_COLUMN}}" 
FROM {{.ENTITY_TABLE}}
	{{.JOIN}} {{.LABELS_TABLE}}
		ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
{{if or .hasFieldCriteria .hasLabelCriteria}}
WHERE {{.ENTITY_TABLE}}.paging_sequence IN 
	(SELECT matching_resources.paging_sequence FROM matching_resources)
{{end}}
{{.ORDER_BY}}
{{.FOR_UPDATE_OF}};`

const SelectNoLabelsQueryTemplate = `
{{if or .hasFieldCriteria .hasLabelCriteria}}
WITH matching_resources as (SELECT DISTINCT {{.ENTITY_TABLE}}.paging_sequence
							FROM {{.ENTITY_TABLE}}
							{{if .hasLabelCriteria}}
							{{.JOIN}} {{.LABELS_TABLE}} 
								ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
							{{end}}
							{{.WHERE}}
							{{.ORDER_BY_SEQUENCE}}
							{{.LIMIT}})
{{end}}
SELECT *
FROM {{.ENTITY_TABLE}}
{{if or .hasFieldCriteria .hasLabelCriteria}}
WHERE {{.ENTITY_TABLE}}.paging_sequence IN 
	(SELECT matching_resources.paging_sequence FROM matching_resources)
{{end}}
{{.ORDER_BY}}
{{.FOR_UPDATE_OF}};`

const DeleteQueryTemplate = `
DELETE FROM {{.ENTITY_TABLE}}
{{if .hasLabelCriteria}}
	USING (SELECT {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}}
			FROM {{.ENTITY_TABLE}}
				{{.JOIN}} {{.LABELS_TABLE}}
					ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
			{{.WHERE}}) t
	WHERE {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = t.{{.PRIMARY_KEY}}
{{else if .hasFieldCriteria}}
	{{.WHERE}}
{{end}}
{{.RETURNING}};`

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
	return &pgQuery{
		labelEntity:       entity.LabelEntity(),
		entityTableName:   entity.TableName(),
		entityTags:        getDBTags(entity, nil),
		labelEntityTags:   getDBTags(entity.LabelEntity(), nil),
		db:                qb.db,
		fieldsWhereClause: &whereClauseTree{},
		labelsWhereClause: &whereClauseTree{
			sqlBuilder: &treeSqlBuilder{
				buildSQL: func(childrenSQL []string) string {
					return fmt.Sprintf("(%s IN (%s))", entity.LabelEntity().ReferenceColumn(), strings.Join(childrenSQL, fmt.Sprintf(" %s ", INTERSECT)))
				},
			},
		},
	}
}

type orderRule struct {
	field     string
	orderType query.OrderType
}

// pgQuery is used to construct postgres queries. It should be constructed only via the query builder. It is not safe for concurrent use.
type pgQuery struct {
	db              pgDB
	labelEntity     PostgresLabel
	entityTags      []tagType
	labelEntityTags []tagType

	queryParams []interface{}

	orderByFields   []orderRule
	hasLock         bool
	limit           string
	returningFields []string
	entityTableName string

	fieldsWhereClause *whereClauseTree
	labelsWhereClause *whereClauseTree
	shouldRebind      bool
	err               error
}

func (pq *pgQuery) List(ctx context.Context) (*sqlx.Rows, error) {
	q, err := pq.resolveQueryTemplate(ctx, SelectQueryTemplate)
	if err != nil {
		return nil, err
	}
	return pq.db.QueryxContext(ctx, q, pq.queryParams...)
}

func (pq *pgQuery) ListNoLabels(ctx context.Context) (*sqlx.Rows, error) {
	q, err := pq.resolveQueryTemplate(ctx, SelectNoLabelsQueryTemplate)
	if err != nil {
		return nil, err
	}
	return pq.db.QueryxContext(ctx, q, pq.queryParams...)
}

func (pq *pgQuery) Query(ctx context.Context, queryName storage.NamedQuery, queryParams map[string]interface{}) (*sqlx.Rows, error) {
	sql, err := util.Tsprintf(storage.GetNamedQuery(queryName), pq.getTemplateParams())
	if err != nil {
		return nil, err
	}
	sql, args, err := sqlx.Named(sql, queryParams)
	if err != nil {
		return nil, err
	}
	sql, args, err = sqlx.In(sql, args...)
	if err != nil {
		return nil, err
	}
	sql = pq.db.Rebind(sql)

	return pq.db.QueryxContext(ctx, sql, args...)
}

func (pq *pgQuery) Count(ctx context.Context) (int, error) {
	q, err := pq.resolveQueryTemplate(ctx, CountQueryTemplate)
	if err != nil {
		return 0, err
	}
	var count int
	if err := pq.db.GetContext(ctx, &count, q, pq.queryParams...); err != nil {
		return 0, err
	}
	return count, nil
}

func (pq *pgQuery) CountLabelValues(ctx context.Context) (int, error) {
	q, err := pq.resolveQueryTemplate(ctx, CountLabelValuesQueryTemplate)
	if err != nil {
		return 0, err
	}
	var count int
	if err := pq.db.GetContext(ctx, &count, q, pq.queryParams...); err != nil {
		return 0, err
	}
	return count, nil
}

func (pq *pgQuery) Delete(ctx context.Context) (sql.Result, error) {
	q, err := pq.resolveQueryTemplate(ctx, DeleteQueryTemplate)
	if err != nil {
		return nil, err
	}

	result, err := pq.db.ExecContext(ctx, q, pq.queryParams...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (pq *pgQuery) DeleteReturning(ctx context.Context, fields ...string) (*sqlx.Rows, error) {
	if err := validateReturningFields(columnsByTags(pq.entityTags), fields...); err != nil {
		return nil, err
	}
	pq.returningFields = append(pq.returningFields, fields...)

	q, err := pq.resolveQueryTemplate(ctx, DeleteQueryTemplate)
	if err != nil {
		return nil, err
	}

	rows, err := pq.db.QueryxContext(ctx, q, pq.queryParams...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (pq *pgQuery) resolveQueryTemplate(ctx context.Context, template string) (string, error) {
	if pq.err != nil {
		return "", pq.err
	}
	if pq.labelEntity == nil {
		return "", fmt.Errorf("query builder requires the entity to have associated label entity")
	}
	data := pq.getTemplateParams()

	q, err := util.Tsprintf(template, data)
	if err != nil {
		return "", err
	}
	if q, err = pq.finalizeSQL(ctx, q); err != nil {
		return "", err
	}

	return q, nil
}

func (pq *pgQuery) getTemplateParams() map[string]interface{} {
	hasFieldCriteria := len(pq.fieldsWhereClause.children) != 0 || len(pq.limit) != 0
	hasLabelCriteria := len(pq.labelsWhereClause.children) != 0
	data := map[string]interface{}{
		"hasFieldCriteria":  hasFieldCriteria,
		"hasLabelCriteria":  hasLabelCriteria,
		"ENTITY_TABLE":      pq.entityTableName,
		"PRIMARY_KEY":       PrimaryKeyColumn,
		"LABELS_TABLE":      pq.labelEntity.LabelsTableName(),
		"REF_COLUMN":        pq.labelEntity.ReferenceColumn(),
		"JOIN":              pq.joinSQL(),
		"WHERE":             pq.whereSQL(),
		"FOR_UPDATE_OF":     pq.lockSQL(),
		"ORDER_BY":          pq.orderBySQL(),
		"ORDER_BY_SEQUENCE": pq.orderBySequenceSQL(),
		"LIMIT":             pq.limitSQL(),
		"RETURNING":         pq.returningSQL(),
	}
	return data
}

func (pq *pgQuery) WithCriteria(criteria ...query.Criterion) *pgQuery {
	if pq.err != nil {
		return pq
	}
	labelQueryCount := 0
	var builder = *defaultTreeSqlBuilder
	var subSelectStatementFunc = func(childrenSQL []string) string {
		return fmt.Sprintf("(SELECT %s FROM %s WHERE (%s))", pq.labelEntity.ReferenceColumn(), pq.labelEntity.LabelsTableName(), strings.Join(childrenSQL, fmt.Sprintf(" %s ", AND)))
	}
	for _, criterion := range criteria {
		if err := criterion.Validate(); err != nil {
			pq.err = err
			return pq
		}
		if hasMultiVariateOp(criteria) {
			pq.shouldRebind = true
		}
		switch criterion.Type {
		case query.FieldQuery:
			columns := columnsByTags(pq.entityTags)
			columnName := criterion.LeftOp
			if strings.Contains(columnName, "/") {
				columnName = strings.Split(columnName, "/")[0]
				ttype := findTagType(pq.entityTags, columnName)
				if ttype != jsonType {
					pq.err = &util.UnsupportedQueryError{Message: fmt.Sprintf("unsupported field query: json notation on non json column: %s", columnName)}
					return pq
				}
			}
			if !columns[columnName] {
				pq.err = &util.UnsupportedQueryError{Message: fmt.Sprintf("unsupported field query key: %s", criterion.LeftOp)}
				return pq
			}
			pq.fieldsWhereClause.children = append(pq.fieldsWhereClause.children, &whereClauseTree{
				criterion: criterion,
				dbTags:    pq.entityTags,
				tableName: pq.entityTableName,
			})
		case query.LabelQuery:
			labelQueryCount++
			if labelQueryCount > 1 { // 2 or more labelQueries need to be intersected
				builder.buildSQL = subSelectStatementFunc
			}
			pq.labelsWhereClause.children = append(pq.labelsWhereClause.children, &whereClauseTree{
				children: []*whereClauseTree{
					{
						criterion: query.ByField(query.EqualsOperator, "key", criterion.LeftOp),
						dbTags:    pq.labelEntityTags,
					},
					{
						criterion: query.ByField(criterion.Operator, "val", criterion.RightOp...),
						dbTags:    pq.labelEntityTags,
					},
				},
				sqlBuilder: &builder,
			})
		case query.ResultQuery:
			pq.processResultCriteria(criterion)

		case query.ExistQuery:
			pq.fieldsWhereClause.children = append(pq.fieldsWhereClause.children, &whereClauseTree{
				criterion: criterion,
				dbTags:    pq.entityTags,
				tableName: pq.entityTableName,
			})
		}
	}

	return pq
}

func (pq *pgQuery) WithLock() *pgQuery {
	if pq.err != nil {
		return pq
	}
	if _, ok := pq.db.(*sqlx.Tx); ok {
		pq.hasLock = true
	}
	return pq
}

func (pq *pgQuery) limitSQL() string {
	if len(pq.limit) > 0 {
		pq.queryParams = append(pq.queryParams, pq.limit)
		return "LIMIT ?"
	}
	return ""
}

// joinSQL is a performance improvement - queries can work with LEFT JOIN always but are slower
// JOIN is used when a label query is present, LEFT JOIN is used when no label query is present so that resultset includes
// unlabelled resources
func (pq *pgQuery) joinSQL() string {
	if len(pq.labelsWhereClause.children) == 0 {
		return "LEFT JOIN"
	}
	return "JOIN"
}

func (pq *pgQuery) whereSQL() string {
	whereClause := &whereClauseTree{
		children: []*whereClauseTree{
			pq.fieldsWhereClause,
			pq.labelsWhereClause,
		},
	}
	whereSQL, queryParams := whereClause.compileSQL()
	if len(whereSQL) == 0 {
		return ""
	}
	pq.queryParams = append(pq.queryParams, queryParams...)
	return fmt.Sprintf(" WHERE %s", whereSQL)
}

func (pq *pgQuery) returningSQL() string {
	fieldsCount := len(pq.returningFields)
	switch fieldsCount {
	case 0:
		return ""
	default:
		return " RETURNING " + strings.Join(pq.returningFields, ", ")
	}
}

func (pq *pgQuery) lockSQL() string {
	if pq.hasLock {
		// Lock the rows if we are in transaction so that update operations on those rows can rely on unchanged data
		// This allows us to handle concurrent updates on the same rows by executing them sequentially as
		// before updating we have to anyway select the rows and can therefore lock them
		return fmt.Sprintf("FOR UPDATE OF %s", pq.entityTableName)
	}
	return ""
}

func (pq *pgQuery) finalizeSQL(ctx context.Context, sql string) (string, error) {
	if pq.shouldRebind {
		var err error
		// sqlx.In requires question marks(?) instead of positional arguments (the ones pgsql uses) in order to map the list argument to the IN operation
		if sql, pq.queryParams, err = sqlx.In(sql, pq.queryParams...); err != nil {
			return "", err
		}
	}
	sql = pq.db.Rebind(sql)

	newline := regexp.MustCompile("\n\n")
	sql = newline.ReplaceAllString(sql, "\n")

	space := regexp.MustCompile(`\s+`)
	sql = space.ReplaceAllString(sql, " ")
	sql = strings.TrimSpace(sql)
	log.C(ctx).Debugf("Executing postgres query: %s", sql)

	return sql, nil
}

func (pq *pgQuery) processResultCriteria(c query.Criterion) *pgQuery {
	if c.Type != query.ResultQuery {
		pq.err = fmt.Errorf("result query is expected, but %s is provided", c.Type)
		return pq
	}
	switch c.LeftOp {
	case query.OrderBy:
		rule := orderRule{
			field:     c.RightOp[0],
			orderType: query.OrderType(c.RightOp[1]),
		}
		if err := validateOrderFields(columnsByTags(pq.entityTags), rule); err != nil {
			pq.err = err
			return pq
		}
		pq.orderByFields = append(pq.orderByFields, rule)
	case query.Limit:
		if pq.limit != "" {
			pq.err = fmt.Errorf("zero/one limit expected but multiple provided")
			return pq
		}
		pq.limit = c.RightOp[0]
	}
	return pq
}

func (pq *pgQuery) orderBySQL() string {
	sql := ""
	rules := pq.orderByFields
	if len(rules) > 0 {
		sql += "ORDER BY"
		for _, orderRule := range rules {
			sql += fmt.Sprintf(" %s %s,", orderRule.field, orderRule.orderType)
		}
		sql = sql[:len(sql)-1]
	}

	if sql == "" {
		sql = fmt.Sprintf("ORDER BY %s.paging_sequence ASC", pq.entityTableName)
	}
	return sql
}

func (pq *pgQuery) orderBySequenceSQL() string {
	if len(pq.limit) > 0 {
		return fmt.Sprintf("ORDER BY %s.paging_sequence ASC", pq.entityTableName)
	}
	return ""
}

func validateOrderFields(columns map[string]bool, orderRules ...orderRule) error {
	fields := make([]string, 0, len(orderRules))
	orderTypes := make([]string, 0, len(orderRules))
	for _, or := range orderRules {
		fields = append(fields, or.field)
		orderTypes = append(orderTypes, string(or.orderType))
	}
	if err := validateFields(map[string]bool{
		string(query.AscOrder):  true,
		string(query.DescOrder): true,
	}, "unsupported order type: %s", orderTypes...); err != nil {
		return err
	}
	return validateFields(columns, "unsupported entity field for order by: %s", fields...)
}

func validateReturningFields(columns map[string]bool, returningFields ...string) error {
	if len(returningFields) == 0 {
		return fmt.Errorf("returning fields cannot be empty")
	}
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
