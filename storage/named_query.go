package storage

type NamedQuery int

const (
	QueryByMissingLabel NamedQuery = iota
	QueryByExistingLabel
	QueryForLastOperationsPerResource
	QueryForLabelLessVisibilities
	QueryForLabelLessPlanVisibilities
)

var namedQueries = map[NamedQuery]string{

	QueryByMissingLabel: `
	SELECT {{.ENTITY_TABLE}}.*,
	{{.LABELS_TABLE}}.id         "{{.LABELS_TABLE}}.id",
	{{.LABELS_TABLE}}.key        "{{.LABELS_TABLE}}.key",
	{{.LABELS_TABLE}}.val        "{{.LABELS_TABLE}}.val",
	{{.LABELS_TABLE}}.created_at "{{.LABELS_TABLE}}.created_at",
	{{.LABELS_TABLE}}.updated_at "{{.LABELS_TABLE}}.updated_at",
	{{.LABELS_TABLE}}.{{.REF_COLUMN}} "{{.LABELS_TABLE}}.{{.REF_COLUMN}}" 
	FROM {{.ENTITY_TABLE}}
		{{.JOIN}} {{.LABELS_TABLE}}
		ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
	WHERE NOT EXISTS
	(SELECT ID FROM {{.LABELS_TABLE}} 
				WHERE key=:key
				AND {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}})`,
	QueryByExistingLabel: `
	SELECT {{.ENTITY_TABLE}}.*,
	{{.LABELS_TABLE}}.id         "{{.LABELS_TABLE}}.id",
	{{.LABELS_TABLE}}.key        "{{.LABELS_TABLE}}.key",
	{{.LABELS_TABLE}}.val        "{{.LABELS_TABLE}}.val",
	{{.LABELS_TABLE}}.created_at "{{.LABELS_TABLE}}.created_at",
	{{.LABELS_TABLE}}.updated_at "{{.LABELS_TABLE}}.updated_at",
	{{.LABELS_TABLE}}.{{.REF_COLUMN}} "{{.LABELS_TABLE}}.{{.REF_COLUMN}}" 
	FROM {{.ENTITY_TABLE}}
		{{.JOIN}} {{.LABELS_TABLE}}
		ON {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}}
	WHERE EXISTS
	(SELECT ID FROM {{.LABELS_TABLE}} 
				WHERE key=:key
				AND {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}})`,
	QueryForLastOperationsPerResource: `
	SELECT {{.ENTITY_TABLE}}.*
	FROM {{.ENTITY_TABLE}} 
    INNER JOIN
		 (
			 SELECT max({{.ENTITY_TABLE}}.paging_sequence) paging_sequence
			 FROM {{.ENTITY_TABLE}}
			 GROUP BY resource_id
		 ) LAST_OPERATIONS
		 ON {{.ENTITY_TABLE}}.paging_sequence = LAST_OPERATIONS.paging_sequence
	WHERE resource_id IN (:id_list)`,
	QueryForLabelLessVisibilities: `
	SELECT v.* FROM visibilities v
	LEFT OUTER JOIN visibility_labels vl on v.id = vl.visibility_id
	WHERE (vl.id IS NULL and v.platform_id in (:platform_ids)) OR v.platform_id IS NULL`,
	QueryForLabelLessPlanVisibilities: `
	SELECT v.* FROM visibilities v
	LEFT OUTER JOIN visibility_labels vl on v.id = vl.visibility_id
	WHERE (vl.id IS NULL and v.service_plan_id in (:service_plan_ids))`,
}

func GetNamedQuery(query NamedQuery) string {
	return namedQueries[query]
}
