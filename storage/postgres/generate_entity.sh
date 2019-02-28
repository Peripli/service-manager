#!/bin/sh

TYPE=$1
TYPE_LOWER=$(echo "$TYPE" | tr '[:upper:]' '[:lower:]')
[[ "$2" == "Labels" ]] && SUPPORTS_LABELS=true || SUPPORTS_LABELS=false
[[ "$3" != "" ]] && TABLE_NAME=$3 || TABLE_NAME="${TYPE_LOWER}s"

ROWS_TO_LIST_BODY=""
LABELS_BODY=""
LABELS_FUNCS=""
OPTIONAL_IMPORTS="
	\"database/sql\"
	\"fmt\"
    \"time\"
    \"github.com/gofrs/uuid\"
	"
if $SUPPORTS_LABELS ; then
	ROWS_TO_LIST_BODY="entities := make(map[string]*types.${TYPE})
	labels := make(map[string]map[string][]string)
	result := &types.${TYPE}s{
		${TYPE}s: make([]*types.${TYPE}, 0),
	}
	for rows.Next() {
		row := struct {
			*${TYPE}
			*${TYPE}Label \`db:"${TYPE_LOWER}_labels"\`
		}{}
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		entity, ok := entities[row.${TYPE}.ID]
		if !ok {
			entity = row.${TYPE}.ToObject().(*types.${TYPE})
			entities[row.${TYPE}.ID] = entity
			result.${TYPE}s = append(result.${TYPE}s, entity)
		}
		if labels[entity.ID] == nil {
			labels[entity.ID] = make(map[string][]string)
		}
		labels[entity.ID][row.${TYPE}Label.Key.String] = append(labels[entity.ID][row.${TYPE}Label.Key.String], row.${TYPE}Label.Val.String)
	}

	for _, b := range result.${TYPE}s {
		b.Labels = labels[b.ID]
	}
	return result, nil"

LABELS_BODY="return ${TYPE_LOWER}Labels{}"

LABELS_FUNCS="$(cat <<-EOF
type ${TYPE}Label struct {
	ID        sql.NullString \`db:"id"\`
	Key       sql.NullString \`db:"key"\`
	Val       sql.NullString \`db:"val"\`
	CreatedAt *time.Time     \`db:"created_at"\`
	UpdatedAt *time.Time     \`db:"updated_at"\`
	${TYPE}ID  sql.NullString \`db:"${TYPE_LOWER}_id"\`
}

func (el ${TYPE}Label) TableName() string {
	return "${TYPE_LOWER}_labels"
}

func (el ${TYPE}Label) PrimaryColumn() string {
	return "id"
}

func (el ${TYPE}Label) ReferenceColumn() string {
	return "${TYPE_LOWER}_id"
}

func (el ${TYPE}Label) Empty() Label {
	return ${TYPE}Label{}
}

func (el ${TYPE}Label) New(entityID, id, key, value string) Label {
	now := time.Now()
	return ${TYPE}Label{
		ID:        toNullString(id),
		Key:       toNullString(key),
		Val:       toNullString(value),
		${TYPE}ID:  toNullString(entityID),
		CreatedAt: &now,
		UpdatedAt: &now,
	}
}

func (el ${TYPE}Label) GetKey() string {
	return el.Key.String
}

func (el ${TYPE}Label) GetValue() string {
	return el.Val.String
}

type ${TYPE_LOWER}Labels []*${TYPE}Label

func (el ${TYPE_LOWER}Labels) Single() Label {
	return &${TYPE}Label{}
}

func (el ${TYPE_LOWER}Labels) FromDTO(entityID string, labels types.Labels) ([]Label, error) {
	var result []Label
	now := time.Now()
	for key, values := range labels {
		for _, labelValue := range values {
			UUID, err := uuid.NewV4()
			if err != nil {
				return nil, fmt.Errorf("could not generate GUID for broker label: %s", err)
			}
			id := UUID.String()
			bLabel := &${TYPE}Label{
				ID:        toNullString(id),
				Key:       toNullString(key),
				Val:       toNullString(labelValue),
				CreatedAt: &now,
				UpdatedAt: &now,
				${TYPE}ID:  toNullString(entityID),
			}
			result = append(result, bLabel)
		}
	}
	return result, nil
}

func (els ${TYPE_LOWER}Labels) ToDTO() types.Labels {
	labelValues := make(map[string][]string)
	for _, label := range els {
		values, exists := labelValues[label.Key.String]
		if exists {
			labelValues[label.Key.String] = append(values, label.Val.String)
		} else {
			labelValues[label.Key.String] = []string{label.Val.String}
		}
	}
	return labelValues
}
EOF
)"
else
LABELS_BODY="return nil"
ROWS_TO_LIST_BODY="result := &types.${TYPE}s{}
	for rows.Next() {
		var item ${TYPE}
		if err := rows.StructScan(&item); err != nil {
			return nil, err
		}
		result.Add(item.ToObject())
	}
	return result, nil"
OPTIONAL_IMPORTS=""
fi

cat > ${TYPE_LOWER}_gen.go <<EOL
// GENERATED. DO NOT MODIFY!

package postgres

import (
	"github.com/jmoiron/sqlx"

	"github.com/Peripli/service-manager/pkg/types"
	${OPTIONAL_IMPORTS}
)

func (${TYPE}) Empty() Entity {
	return ${TYPE}{}
}

func (${TYPE}) PrimaryColumn() string {
	return "id"
}

func (${TYPE}) TableName() string {
	return "${TABLE_NAME}"
}

func (e ${TYPE}) GetID() string {
	return e.ID
}

func (e ${TYPE}) Labels() EntityLabels {
    ${LABELS_BODY}
}

func (e ${TYPE}) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
    ${ROWS_TO_LIST_BODY}
}
${LABELS_FUNCS}
EOL