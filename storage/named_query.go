package storage

type NamedQuery int

const (
	QueryByMissingLabel NamedQuery = iota
	QueryByExistingLabel
	QueryForLastOperationsPerResource
	QueryForLabelLessVisibilities
	QueryForPlatformVisibility
	QueryForVisibilityLabelByPlatform
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
	QueryForPlatformVisibility: `select  * from {{.ENTITY_TABLE}} inner join public.visibility_labels on visibilities.id = visibility_labels.visibility_id
where key ='subaccount_id' and val ='ab8b7ee1-4806-424f-b0dc-8b3256b7a501' and platform_id =  '(:platform_id)' and service_plan_id ='(:service_plan_id)`,
	QueryForVisibilityLabelByPlatform: `select 1 as label_key_found from visibilities inner join visibility_labels on visibilities.id = visibility_labels.visibility_id
where platform_id='service-manager' and key ='subaccount_id' limit 1`,
}

func GetNamedQuery(query NamedQuery) string {
	return namedQueries[query]
}
