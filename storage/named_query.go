package storage

type NamedQuery int



const (
	QueryByMissingLabel NamedQuery = iota
)

var namedQueries =  map[NamedQuery]string {
	QueryByMissingLabel : `
	SELECT * FROM {{.ENTITY_TABLE}}
	WHERE NOT EXISTS
	(SELECT ID FROM {{.LABELS_TABLE}} 
				WHERE key=:key
				AND {{.ENTITY_TABLE}}.{{.PRIMARY_KEY}} = {{.LABELS_TABLE}}.{{.REF_COLUMN}})`,
}

func GetNamedQuery(query NamedQuery) string {
return namedQueries[query]
}

