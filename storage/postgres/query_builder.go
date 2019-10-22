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

const PrimaryKeyColumn = "id"
const SELECTColumns = `
{{.ENTITY_TABLE}}.*,
{{.LABELS_TABLE}}.id         "{{.LABELS_TABLE}}.id",
{{.LABELS_TABLE}}.key        "{{.LABELS_TABLE}}.key",
{{.LABELS_TABLE}}.val        "{{.LABELS_TABLE}}.val",
{{.LABELS_TABLE}}.created_at "{{.LABELS_TABLE}}.created_at",
{{.LABELS_TABLE}}.updated_at "{{.LABELS_TABLE}}.updated_at",
{{.LABELS_TABLE}}.{{.REF_COLUMN}} "{{.LABELS_TABLE}}.{{.REF_COLUMN}}"`
const COUNTColumns = `COUNT(DISTINCT {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}})`

const SELECTWithoutCriteriaTemplate = `
SELECT %s
FROM {{.ENTITY_TABLE}}
         LEFT JOIN {{.LABELS_TABLE}}
                   ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
{{.FOR_SHARE_OF}}
{{.ORDER_BY}};`

const SELECTWithCriteriaTemplate = `
WITH matching_resources as (SELECT DISTINCT {{.ENTITY_TABLE}}.paging_sequence
                          FROM {{.ENTITY_TABLE}}
                                   LEFT JOIN {{.LABELS_TABLE}} 
										ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
                          {{.WHERE}}
                          {{.ORDER_BY_SEQUENCE}}
                          {{.LIMIT}})
SELECT %s 
FROM {{.ENTITY_TABLE}}
         LEFT JOIN {{.LABELS_TABLE}}
                   ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
WHERE {{.ENTITY_TABLE}}.paging_sequence IN 
		(SELECT matching_resources.paging_sequence FROM matching_resources)
{{.FOR_SHARE_OF}}
{{.ORDER_BY}};`

const DELETEWithoutCriteriaTemplate = `
DELETE 
FROM {{.ENTITY_TABLE}}
{{.RETURNING}};`

const DELETEWithCriteriaTemplate = `
WITH matching_resources as (SELECT DISTINCT {{.ENTITY_TABLE}}.paging_sequence
                          FROM {{.ENTITY_TABLE}}
                                   LEFT JOIN {{.LABELS_TABLE}}
										ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
                          {{.WHERE}})
DELETE FROM {{.ENTITY_TABLE}}
WHERE {{.ENTITY_TABLE}}.paging_sequence IN 
		(SELECT matching_resources.paging_sequence FROM matching_resources)
{{.RETURNING}};`

const noCriteria = "nc"
const withCriteria = "c"

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
		labelEntityTags: getDBTags(entity.LabelEntity(), nil),
		db:              qb.db,
		fieldsWhereClause: &whereClauseTree{
			operator: AND,
		},
		labelsWhereClause: &whereClauseTree{
			operator: OR,
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
	if pq.err != nil {
		return nil, pq.err
	}

	templates := map[string]string{
		noCriteria:   fmt.Sprintf(SELECTWithoutCriteriaTemplate, SELECTColumns),
		withCriteria: fmt.Sprintf(SELECTWithCriteriaTemplate, SELECTColumns),
	}
	q, err := pq.resolveQueryTemplate(ctx, templates)
	if err != nil {
		return nil, err
	}
	return pq.db.QueryxContext(ctx, q, pq.queryParams...)
}

func (pq *pgQuery) Count(ctx context.Context) (int, error) {
	if pq.err != nil {
		return 0, pq.err
	}
	templates := map[string]string{
		noCriteria:   fmt.Sprintf(SELECTWithoutCriteriaTemplate, COUNTColumns),
		withCriteria: fmt.Sprintf(SELECTWithCriteriaTemplate, COUNTColumns),
	}
	q, err := pq.resolveQueryTemplate(ctx, templates)
	if err != nil {
		return 0, err
	}
	var count int
	if err := pq.db.GetContext(ctx, &count, q, pq.queryParams...); err != nil {
		return 0, err
	}
	return count, nil
}

func (pq *pgQuery) Delete(ctx context.Context) (*sqlx.Rows, error) {
	if pq.err != nil {
		return nil, pq.err
	}
	templates := map[string]string{
		noCriteria:   DELETEWithoutCriteriaTemplate,
		withCriteria: DELETEWithCriteriaTemplate,
	}
	q, err := pq.resolveQueryTemplate(ctx, templates)
	if err != nil {
		return nil, err
	}
	return pq.db.QueryxContext(ctx, q, pq.queryParams...)
}

func (pq *pgQuery) resolveQueryTemplate(ctx context.Context, templates map[string]string) (string, error) {
	if pq.labelEntity == nil {
		return "", fmt.Errorf("query builder requires the entity to have associated label entity")
	}
	data := map[string]interface{}{
		"ENTITY_TABLE":      pq.entityTableName,
		"PRIMARY_KEY":       PrimaryKeyColumn,
		"LABELS_TABLE":      pq.labelEntity.LabelsTableName(),
		"REF_COLUMN":        pq.labelEntity.ReferenceColumn(),
		"WHERE":             pq.whereSQL(),
		"FOR_SHARE_OF":      pq.lockSQL(),
		"ORDER_BY":          pq.orderBySQL(),
		"ORDER_BY_SEQUENCE": pq.orderBySequenceSQL(),
		"LIMIT":             pq.limitSQL(),
		"RETURNING":         pq.returningSQL(),
	}

	var q string
	hasCriteria := len(pq.fieldsWhereClause.children) != 0 ||
		len(pq.labelsWhereClause.children) != 0 ||
		len(pq.limit) != 0

	if hasCriteria {
		q = templates[withCriteria]
	} else {
		q = templates[noCriteria]
	}
	var err error
	q, err = tsprintf(q, data)
	if err != nil {
		return "", err
	}
	if q, err = pq.finalizeSQL(ctx, q); err != nil {
		return "", err
	}

	return q, nil
}

func (pq *pgQuery) Return(fields ...string) *pgQuery {
	if pq.err != nil {
		return pq
	}
	if err := validateReturningFields(columnsByTags(pq.entityTags), fields...); err != nil {
		pq.err = err
		return pq
	}
	pq.returningFields = append(pq.returningFields, fields...)

	return pq
}

func (pq *pgQuery) WithCriteria(criteria ...query.Criterion) *pgQuery {
	if pq.err != nil {
		return pq
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
			if !columns[criterion.LeftOp] {
				pq.err = &util.UnsupportedQueryError{Message: fmt.Sprintf("unsupported field query key: %s", criterion.LeftOp)}
				return pq
			}
			pq.fieldsWhereClause.children = append(pq.fieldsWhereClause.children, &whereClauseTree{
				criterion: criterion,
				dbTags:    pq.entityTags,
				tableName: pq.entityTableName,
			})
		case query.LabelQuery:
			pq.labelsWhereClause.children = append(pq.labelsWhereClause.children, &whereClauseTree{
				operator: AND,
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
			})
		case query.ResultQuery:
			pq.processResultCriteria(criterion)
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

func (pq *pgQuery) whereSQL() string {
	whereClause := &whereClauseTree{
		operator: AND,
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
		return fmt.Sprintf("FOR SHARE OF %s", pq.entityTableName)
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
