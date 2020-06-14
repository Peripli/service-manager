package storage

type NamedQuery int

const (
	QueryByMissingLabel NamedQuery = iota
	QueryByExistingLabel
	QueryForLastOperationsPerResource
	CleanOperations
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
	QueryForLastOperationsPerResource:`
	SELECT id,resource_id,ops.state,type,errors,external_id,description,updated_at,created_at,deletion_scheduled,reschedule_timestamp
	FROM operations ops 
    inner join
		 (
			 select max(op.paging_sequence) last_operation_sequence
			 from operations op
			 group by resource_id
		 ) lastOperationPerResource
		 on lastOperationPerResource.last_operation_sequence = ops.paging_sequence
	WHERE resource_id in ({{.RESOURCE_IDS}})`,
	CleanOperations:`
	SELECT id,resource_id,ops.state,type,errors,external_id,description,updated_at,created_at,deletion_scheduled,reschedule_timestamp
	FROM operations ops 
    left join
		 (
			 select max(op.paging_sequence) last_operation_sequence
			 from operations op
			 group by resource_id
		 ) lastOperationPerResource
		 on lastOperationPerResource.last_operation_sequence= ops.paging_sequence
	WHERE resource_id in ({{.RESOURCE_IDS}})
	and lastOperationPerResource.last_operation_sequence is null`,
}

func GetNamedQuery(query NamedQuery) string {
	return namedQueries[query]
}
