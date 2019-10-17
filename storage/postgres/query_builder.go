package postgres

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/jmoiron/sqlx"
)

const SELECTWithoutLabelsAndWithoutCriteriaTemplate = `
SELECT %s 
FROM {{.ENTITY_TABLE}} t
{{.FOR_SHARE_OF}};`

const SELECTWithLabelsAndWithoutCriteriaTemplate = `
SELECT %s 
FROM {{.ENTITY_TABLE}} t
         {{.JOIN}} {{.LABELS_TABLE}}
                   ON t.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.LABELS_TABLE_REF_COLUMN}}
{{.FOR_SHARE_OF}};`

const SELECTWithoutLabelsAndWithCriteriaTemplate = `
WITH matching_resources as (SELECT DISTINCT t.paging_sequence
                          FROM {{.ENTITY_TABLE}} t
                          {{.WHERE}}
                          {{.ORDER_BY}}
                          {{.LIMIT}})
SELECT %s 
FROM {{.ENTITY_TABLE}} t
WHERE t.paging_sequence IN 
		(SELECT matching_resources.paging_sequence FROM matching_resources)
{{.FOR_SHARE_OF}}
{{.ORDER_BY}};`

const SELECTWithLabelsAndWithCriteriaTemplate = `
WITH matching_resources as (SELECT DISTINCT t.paging_sequence
                          FROM {{.ENTITY_TABLE}} t
                                   {{.JOIN}} {{.LABELS_TABLE}} 
										ON t.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.LABELS_TABLE_REF_COLUMN}}
                          {{.WHERE}}
                          {{.ORDER_BY}}
                          {{.LIMIT}})
SELECT %s 
FROM {{.ENTITY_TABLE}} t
         {{.JOIN}} {{.LABELS_TABLE}}
                   ON t.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.LABELS_TABLE_REF_COLUMN}}
WHERE t.paging_sequence IN 
		(SELECT matching_resources.paging_sequence FROM matching_resources)
{{.FOR_SHARE_OF}}
{{.ORDER_BY}};`

const SELECTWithLabelsColumns = `
t.*,
{{.LABELS_TABLE}}.id         "{{.LABELS_TABLE}}.id",
{{.LABELS_TABLE}}.key        "{{.LABELS_TABLE}}.key",
{{.LABELS_TABLE}}.val        "{{.LABELS_TABLE}}.val",
{{.LABELS_TABLE}}.created_at "{{.LABELS_TABLE}}.created_at",
{{.LABELS_TABLE}}.updated_at "{{.LABELS_TABLE}}.updated_at",
{{.LABELS_TABLE}}.{{.LABELS_TABLE_REF_COLUMN}} "{{.LABELS_TABLE}}.{{.LABELS_TABLE_REF_COLUMN}}"`

const SELECTWithoutLabelsColumns = `t.*`

const COUNTColumns = `COUNT(DISTINCT t.{{.PRIMARY_KEY}})`

const DELETEWithoutCriteriaTemplate = `
DELETE 
FROM {{.ENTITY_TABLE}} t
{{.RETURNING}};`

const DELETEWithCriteriaTemplate = `
WITH matching_resources as (SELECT DISTINCT t.paging_sequence
                          FROM {{.ENTITY_TABLE}} t
                                   {{.JOIN}} {{.LABELS_TABLE}} l
										ON t.{{.PRIMARY_KEY}} = l.{{.LABELS_TABLE_REF_COLUMN}}
                          {{.WHERE}})
DELETE FROM {{.ENTITY_TABLE}} t
WHERE t.paging_sequence IN 
		(SELECT matching_resources.paging_sequence FROM matching_resources)
	AND {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = t.{{.PRIMARY_KEY}}
{{.RETURNING}};`

type orderRule struct {
	field     string
	orderType query.OrderType
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
	return &pgQuery{
		labelEntity:     entity.LabelEntity(),
		entityTableName: entity.TableName(),
		entityTags:      getDBTags(entity, nil),
		db:              qb.db,
		whereClauseTree: &whereClauseTree{
			operator: AND,
		},
	}
}

// pgQuery is used to construct postgres queries. It should be constructed only via the query builder. It is not safe for concurrent use.
type pgQuery struct {
	db              pgDB
	labelEntity     PostgresLabel
	entityTableName string
	entityTags      []tagType

	queryParams []interface{}

	orderByFields   []orderRule
	limit           string
	hasLock         bool
	returningFields []string

	whereClauseTree      *whereClauseTree
	shouldRebind         bool
	labelCriteriaPresent bool
	err                  error
}

func (pgq *pgQuery) List(ctx context.Context) (*sqlx.Rows, error) {
	if pgq.err != nil {
		return nil, pgq.err
	}

	templates := map[string]string{
		"nlnc": fmt.Sprintf(SELECTWithoutLabelsAndWithoutCriteriaTemplate, SELECTWithoutLabelsColumns),
		"lnc":  fmt.Sprintf(SELECTWithLabelsAndWithoutCriteriaTemplate, SELECTWithLabelsColumns),
		"nlc":  fmt.Sprintf(SELECTWithoutLabelsAndWithCriteriaTemplate, SELECTWithoutLabelsColumns),
		"lc":   fmt.Sprintf(SELECTWithLabelsAndWithCriteriaTemplate, SELECTWithLabelsColumns),
	}
	q, err := pgq.resolveQueryTemplate(ctx, templates)
	if err != nil {
		return nil, err
	}
	return pgq.db.QueryxContext(ctx, q, pgq.queryParams...)
}

func (pgq *pgQuery) Count(ctx context.Context) (int, error) {
	if pgq.err != nil {
		return 0, pgq.err
	}
	templates := map[string]string{
		"nlnc": fmt.Sprintf(SELECTWithoutLabelsAndWithoutCriteriaTemplate, COUNTColumns),
		"lnc":  fmt.Sprintf(SELECTWithLabelsAndWithoutCriteriaTemplate, COUNTColumns),
		"nlc":  fmt.Sprintf(SELECTWithoutLabelsAndWithCriteriaTemplate, COUNTColumns),
		"lc":   fmt.Sprintf(SELECTWithLabelsAndWithCriteriaTemplate, COUNTColumns),
	}
	q, err := pgq.resolveQueryTemplate(ctx, templates)
	if err != nil {
		return 0, err
	}
	var count int
	if err := pgq.db.GetContext(ctx, &count, q, pgq.queryParams...); err != nil {
		return 0, err
	}
	return count, nil
}

func (pgq *pgQuery) Delete(ctx context.Context) (*sqlx.Rows, error) {
	if pgq.err != nil {
		return nil, pgq.err
	}
	templates := map[string]string{
		"nlnc": DELETEWithoutCriteriaTemplate,
		"lnc":  DELETEWithoutCriteriaTemplate,
		"nlc":  DELETEWithCriteriaTemplate,
		"lc":   DELETEWithCriteriaTemplate,
	}
	q, err := pgq.resolveQueryTemplate(ctx, templates)
	if err != nil {
		return nil, err
	}
	return pgq.db.QueryxContext(ctx, q, pgq.queryParams...)
}

func (pgq *pgQuery) resolveQueryTemplate(ctx context.Context, templates map[string]string) (string, error) {
	data := map[string]interface{}{
		"ENTITY_TABLE": pgq.entityTableName,
		"PRIMARY_KEY":  "id",
		"JOIN":         pgq.joinSQL(),
		"WHERE":        pgq.whereSQL(),
		"FOR_SHARE_OF": pgq.lockSQL(),
		"ORDER_BY":     pgq.orderBySQL(),
		"LIMIT":        pgq.limitSQL(),
		"RETURNING":    pgq.returningSQL(),
	}

	var q string
	var err error
	hasCriteria := len(pgq.whereClauseTree.children) != 0
	if pgq.labelEntity != nil {
		data["LABELS_TABLE"] = pgq.labelEntity.LabelsTableName()
		data["LABELS_TABLE_REF_COLUMN"] = pgq.labelEntity.ReferenceColumn()
		if hasCriteria {
			q = templates["lc"]
		} else {
			q = templates["lnc"]
		}
	} else {
		if hasCriteria {
			q = templates["nlc"]
		} else {
			q = templates["nlnc"]
		}
	}
	q, err = tsprintf(q, data)
	if err != nil {
		return "", err
	}

	if q, err = pgq.finalizeSQL(ctx, q); err != nil {
		return "", err
	}

	return q, nil
}

func (pgq *pgQuery) Return(fields ...string) *pgQuery {
	if pgq.err != nil {
		return pgq
	}
	if err := validateReturningFields(columnsByTags(pgq.entityTags), fields...); err != nil {
		pgq.err = err
		return pgq
	}
	pgq.returningFields = append(pgq.returningFields, fields...)

	return pgq
}

func (pgq *pgQuery) WithCriteria(criteria ...query.Criterion) *pgQuery {
	if pgq.err != nil {
		return pgq
	}
	if len(criteria) == 0 {
		return pgq
	}
	for _, criterion := range criteria {
		if err := criterion.Validate(); err != nil {
			pgq.err = err
			return pgq
		}
		if hasMultiVariateOp(criteria) {
			pgq.shouldRebind = true
		}
		switch criterion.Type {
		case query.FieldQuery:
			columns := columnsByTags(pgq.entityTags)
			if !columns[criterion.LeftOp] {
				pgq.err = &util.UnsupportedQueryError{Message: fmt.Sprintf("unsupported field query key: %s", criterion.LeftOp)}
				return pgq
			}
			pgq.whereClauseTree.children = append(pgq.whereClauseTree.children, &whereClauseTree{
				criterion: criterion,
			})
		case query.LabelQuery:
			pgq.labelCriteriaPresent = true
			pgq.whereClauseTree.children = append(pgq.whereClauseTree.children, &whereClauseTree{
				operator: OR,
				children: []*whereClauseTree{
					{
						criterion: query.ByField(query.EqualsOperator, "key", criterion.LeftOp),
					},
					{
						criterion: query.ByField(criterion.Operator, "val", criterion.RightOp...),
					},
				},
			})
		case query.ResultQuery:
			pgq.processResultCriteria(criterion)
		}
	}

	return pgq
}

func (pgq *pgQuery) WithLock() *pgQuery {
	if pgq.err != nil {
		return pgq
	}
	if _, ok := pgq.db.(*sqlx.Tx); ok {
		pgq.hasLock = true
	}
	return pgq
}

func (ssq *pgQuery) limitSQL() string {
	if len(ssq.limit) > 0 {
		ssq.queryParams = append(ssq.queryParams, ssq.limit)
		return "LIMIT ?"
	}
	return ""
}

func (ssq *pgQuery) joinSQL() string {
	join := " LEFT JOIN "
	if ssq.labelCriteriaPresent {
		join = " JOIN "
	}
	return join
}

func (ssq *pgQuery) whereSQL() string {
	whereSQL, queryParams, err := ssq.whereClauseTree.compileSQL(ssq.entityTags, "t")
	if err != nil {
		ssq.err = err
		return ""
	}
	ssq.queryParams = append(ssq.queryParams, queryParams...)
	return fmt.Sprintf(" WHERE %s", whereSQL)
}

func (pgq *pgQuery) returningSQL() string {
	for i := range pgq.returningFields {
		if !strings.HasPrefix(pgq.returningFields[i], fmt.Sprintf("%s.", "t")) {
			pgq.returningFields[i] = fmt.Sprintf("%s.%s", "t", pgq.returningFields[i])
		}
	}

	fieldsCount := len(pgq.returningFields)
	switch fieldsCount {
	case 0:
		return ""
	case 1:
		return " RETURNING " + pgq.returningFields[0]
	default:
		return " RETURNING " + strings.Join(pgq.returningFields, ", ")
	}
}

func (pgq *pgQuery) lockSQL() string {
	if pgq.hasLock {
		// Lock the rows if we are in transaction so that update operations on those rows can rely on unchanged data
		// This allows us to handle concurrent updates on the same rows by executing them sequentially as
		// before updating we have to anyway select the rows and can therefore lock them
		return fmt.Sprintf("FOR SHARE OF %s", "t")
	}
	return ""
}

func (pgq *pgQuery) finalizeSQL(ctx context.Context, sql string) (string, error) {
	if pgq.shouldRebind {
		var err error
		// sqlx.In requires question marks(?) instead of positional arguments (the ones pgsql uses) in order to map the list argument to the IN operation
		if sql, pgq.queryParams, err = sqlx.In(sql, pgq.queryParams...); err != nil {
			return "", err
		}
	}
	sql = pgq.db.Rebind(sql)

	space := regexp.MustCompile(`\s+`)
	sql = space.ReplaceAllString(sql, " ")
	sql = strings.TrimSpace(sql)
	log.C(ctx).Debugf("Executing postgres query: %s", sql)

	return sql, nil
}

func (pgq *pgQuery) processResultCriteria(c query.Criterion) *pgQuery {
	if c.Type != query.ResultQuery {
		pgq.err = fmt.Errorf("result query is expected, but %s is provided", c.Type)
		return pgq
	}
	switch c.LeftOp {
	case query.OrderBy:
		rule := orderRule{
			field:     c.RightOp[0],
			orderType: query.OrderType(c.RightOp[1]),
		}
		if err := validateOrderFields(columnsByTags(pgq.entityTags)); err != nil {
			pgq.err = err
			return pgq
		}
		pgq.orderByFields = append(pgq.orderByFields, rule)
	case query.Limit:
		if pgq.limit != "" {
			pgq.err = fmt.Errorf("zero/one limit expected but multiple provided")
			return pgq
		}
		pgq.limit = c.RightOp[0]
	}
	return pgq
}

func (pgq *pgQuery) orderBySQL() string {
	sql := ""
	rules := pgq.orderByFields
	if len(rules) > 0 {
		sql += " ORDER BY"
		for _, orderRule := range rules {
			sql += fmt.Sprintf(" %s %s,", orderRule.field, orderRule.orderType)
		}
		sql = sql[:len(sql)-1]
	}
	return sql
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

// tsprintf stands for "template Sprintf" and fills the specified templated string with the provided data
func tsprintf(tmpl string, data map[string]interface{}) (string, error) {
	t, err := template.New("sql").Parse(tmpl)
	if err != nil {
		return "", err
	}
	buff := &bytes.Buffer{}
	if err := t.Execute(buff, data); err != nil {
		return "", nil
	}
	return buff.String(), nil
}
