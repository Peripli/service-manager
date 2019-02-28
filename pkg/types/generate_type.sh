#!/usr/bin/env bash

TYPE=$1
TYPE_PLURAL="${TYPE}s"
TYPE_LOWER=$(echo "$TYPE" | tr '[:upper:]' '[:lower:]')
TYPE_PLURAL_LOWER=$(echo "$TYPE_PLURAL" | tr '[:upper:]' '[:lower:]')
[[ "$2" == "Labels" ]] && SUPPORTS_LABELS=true || SUPPORTS_LABELS=false

GET_LABELS_BODY="return Labels{}"
WITH_LABELS_BODY="return e"
MARSHAL_JSON="if !e.CreatedAt.IsZero() {
		str := util.ToRFCFormat(e.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !e.UpdatedAt.IsZero() {
		str := util.ToRFCFormat(e.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}"
if $SUPPORTS_LABELS ; then
    GET_LABELS_BODY="return e.Labels"
    WITH_LABELS_BODY="e.Labels = labels
	return e"
	MARSHAL_JSON="$MARSHAL_JSON
	hasNoLabels := true
	for key, values := range e.Labels {
		if key != \"\" && len(values) != 0 {
			hasNoLabels = false
			break
		}
	}
	if hasNoLabels {
		toMarshal.Labels = nil
	}
	"
fi

cat > ${TYPE_LOWER}_gen.go <<EOL
// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/util"
)

type ${TYPE_PLURAL} struct {
	${TYPE_PLURAL} []*${TYPE} \`json:"${TYPE_PLURAL_LOWER}"\`
}

func (e *${TYPE_PLURAL}) Add(object Object) {
	e.${TYPE_PLURAL} = append(e.${TYPE_PLURAL}, object.(*${TYPE}))
}

func (e *${TYPE_PLURAL}) ItemAt(index int) Object {
	return e.${TYPE_PLURAL}[index]
}

func (e *${TYPE_PLURAL}) Len() int {
	return len(e.${TYPE_PLURAL})
}

func (e *${TYPE}) SupportsLabels() bool {
	return ${SUPPORTS_LABELS}
}

func (e *${TYPE}) EmptyList() ObjectList {
	return &${TYPE_PLURAL}{${TYPE_PLURAL}: make([]*${TYPE}, 0)}
}

func (e *${TYPE}) WithLabels(labels Labels) Object {
    ${WITH_LABELS_BODY}
}

func (e *${TYPE}) GetType() ObjectType {
	return ${TYPE}Type
}

func (e *${TYPE}) GetLabels() Labels {
    ${GET_LABELS_BODY}
}

// MarshalJSON override json serialization for http response
func (e *${TYPE}) MarshalJSON() ([]byte, error) {
	type E ${TYPE}
	toMarshal := struct {
		*E
		CreatedAt *string \`json:"created_at,omitempty"\`
		UpdatedAt *string \`json:"updated_at,omitempty"\`
	}{
		E: (*E)(e),
	}
    ${MARSHAL_JSON}
	return json.Marshal(toMarshal)
}

EOL