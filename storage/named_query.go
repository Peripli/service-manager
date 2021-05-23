package storage

type NamedQuery int

const (
	QueryByMissingLabel NamedQuery = iota
	QueryByExistingLabel
	QueryForLastOperationsPerResource
	QueryForLabelLessVisibilities
	QueryForLabelLessPlanVisibilities
	QueryForVisibilityWithPlatformAndPlan
	QueryForPlanByNameAndOfferingsWithVisibility
	QueryForOSBReferenceByPlanSelector
	QueryForSMAAPReferenceByPlanSelector
	QueryForReferenceBySharedInstanceSelector
	QueryForReferenceByNameSelector
	QueryForReferenceByInstanceID
	QueryForReferenceByLabels
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
			 GROUP BY resource_id,resource_type
		 ) LAST_OPERATIONS
		 ON {{.ENTITY_TABLE}}.paging_sequence = LAST_OPERATIONS.paging_sequence
	WHERE resource_id IN (:id_list) AND resource_type = :resource_type`,
	QueryForLabelLessVisibilities: `
	SELECT v.* FROM visibilities v
	WHERE (v.platform_id in (:platform_ids) OR v.platform_id IS NULL) AND
	NOT EXISTS(SELECT vl.id FROM visibility_labels vl WHERE vl.visibility_id = v.id)`,
	QueryForLabelLessPlanVisibilities: `
	SELECT v.* FROM visibilities v
	WHERE (v.service_plan_id in (:service_plan_ids)) AND
	NOT EXISTS(SELECT vl.id FROM visibility_labels vl WHERE vl.visibility_id = v.id)`,
	QueryForVisibilityWithPlatformAndPlan: `
	SELECT v.*
	FROM visibilities v
	WHERE v.service_plan_id = :service_plan_id
	AND (v.platform_id IS NULL
		OR (v.platform_id = :platform_id AND (:key = '' IS TRUE OR NOT EXISTS(SELECT vl.id FROM visibility_labels vl WHERE vl.visibility_id = v.id)))
		OR EXISTS(SELECT vl.id FROM visibility_labels vl WHERE vl.visibility_id = v.id AND vl.key = :key AND vl.val = :val))`,
	QueryForPlanByNameAndOfferingsWithVisibility: `
	SELECT sp.* 
	FROM service_plans sp
		 INNER JOIN service_offerings so
			ON so.id = sp.service_offering_id
		 INNER JOIN visibilities v 
			ON sp.id = v.service_plan_id
		 LEFT OUTER JOIN visibility_labels vl 
			ON v.id = vl.visibility_id
	WHERE (vl.key = :key AND vl.val = :val AND v.platform_id = :platform_id AND sp.catalog_name = :service_plan_name AND so.catalog_name = :service_offering_name)
	OR ((v.platform_id = :platform_id OR v.platform_id IS NULL) 
		AND NOT EXISTS(SELECT vl.visibility_id FROM visibility_labels vl WHERE vl.visibility_id = v.id) 
		AND sp.catalog_name = :service_plan_name AND so.catalog_name = :service_offering_name)`,
	QueryForOSBReferenceByPlanSelector: `
	SELECT service_instances.* FROM service_instances
	INNER JOIN service_plans ON service_plans.ID = service_instances.service_plan_id
	INNER JOIN service_instance_labels ON service_instances.id = service_instance_labels.service_instance_id
	WHERE service_instances.shared=TRUE AND
		  service_plans.CATALOG_NAME = :selector_value AND
		  service_plans.service_offering_id = :offering_id AND
		  service_instance_labels.key = :tenant_identifier AND service_instance_labels.val = :tenant_id`,
	QueryForSMAAPReferenceByPlanSelector: `
	SELECT service_instances.* FROM service_instances
	INNER JOIN service_plans ON service_plans.id = service_instances.service_plan_id
	INNER JOIN service_instance_labels ON service_instances.id = service_instance_labels.service_instance_id
	WHERE service_instances.shared=true AND
		  service_plans.NAME = :selector_value AND
		  service_plans.service_offering_id = :offering_id AND
		  service_instance_labels.key = :tenant_identifier AND service_instance_labels.val = :tenant_id`,
	QueryForReferenceBySharedInstanceSelector: `
	SELECT service_instances.* FROM service_instances
	INNER JOIN service_plans ON service_plans.ID = service_instances.service_plan_id
	INNER JOIN service_instance_labels ON service_instances.ID = service_instance_labels.service_instance_id
	WHERE service_instances.shared=true AND
		  service_plans.service_offering_id = :offering_id AND
		  service_instance_labels.key = :tenant_identifier AND service_instance_labels.val = :tenant_id`,
	QueryForReferenceByNameSelector: `
	SELECT service_instances.* FROM service_instances
	INNER JOIN service_plans ON service_plans.id = service_instances.service_plan_id
	INNER JOIN service_instance_labels ON service_instances.id = service_instance_labels.service_instance_id
	WHERE service_instances.shared=true AND
		  service_instances.name = :selector_value AND
		  service_plans.service_offering_id = :offering_id AND
		  service_instance_labels.key = :tenant_identifier AND service_instance_labels.val = :tenant_id`,
	QueryForReferenceByInstanceID: `
	select service_instances.* from service_instances
	inner join service_plans on service_plans.id = service_instances.service_plan_id
	inner join service_instance_labels on service_instances.id = service_instance_labels.service_instance_id
	where service_instances.id = :selector_value and
		  service_plans.service_offering_id = :offering_id and
		  service_instance_labels.key = :tenant_identifier and service_instance_labels.val = :tenant_id`,
	QueryForReferenceByLabels: `
	select service_instances.* from service_instances
	inner join service_plans on service_plans.id = service_instances.service_plan_id
	inner join service_instance_labels on service_instances.id = service_instance_labels.service_instance_id
	where service_instances.shared=true and
		  service_plans.service_offering_id = :offering_id and
		  service_instance_labels.key = :tenant_identifier and service_instance_labels.val = :tenant_id and
		  service_instance_labels.key = :label_selector_key and service_instance_labels.val = :label_selector_value`,
}

func GetNamedQuery(query NamedQuery) string {
	return namedQueries[query]
}
