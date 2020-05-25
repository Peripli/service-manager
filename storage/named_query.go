package storage

type NamedQuery string

const (

QueryByLabelMissing = `
	SELECT * FROM {{.ENTITY_TABLE}}
	WHERE NOT EXISTS
	(SELECT ID FROM {{.ENTITY_LABELS}} WHERE key=${key} AND {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}})
`
)

