package storage

type NamedQuery string

const (

QueryByLabelMissing = `
	select * from {{.ENTITY}}
	WHERE NOT EXISTS
	(select id from {{.ENTITY_LABELS}} WHERE key=${key} AND {{.ENTITY_LABELS.ref_id}} = {{.ENTITY.id}})
`
)

