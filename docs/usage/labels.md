# Resource labeling and querying

Specific resources in the Service Manager can be labeled in order to be organized into groups relevant to the users. Afterwards, resources can be queried either by the labels attached to them, or by the fields they have.

# Table of Contents

  - [Labels](#labels)
    - [Syntax](#syntax)
  - [Querying](#querying)
    - [Operators](#operator)
    - [Query Types](#query-types)
  - [Supported entities](#supported-entities)
  - [API](#api)

# Labels
Labels are key/value* pairs that can be associated with a resource in the Service Manager.

Service Manager resources can be assigned labels with which to organize them into groups. Labels can be attached to a resource at creation time and added or modified at any time later on. Each label represents a pair of key/values. Each key must be unique for a given resource.

```json
{
    ...
    "labels": [
        {
            "key": "key1",
            "value": ["value1", "value2"]
        },
        {
            "key": "key2",
            "value": ["value3", "value4"]
        }
    ]
}
```

```
* A value in a label represents an array of strings.
```

## Syntax

Valid labels consist of a key and one or multiple values.  
Keys can contain all characteres **except** the query separator (**|**).  
Values can contain all characters, but if any value contains the query separator (**|**), then this symbol must be escaped with a backslash (**\\**) - `This is a value with a \| separator`  
The length of both the key and each value must be between 1 and 255 characters.  

# Querying

Querying can be performed both on labels and resource fields.
A valid query consist one or more criteria and criterion cosists of a left operand, an operator and a right operand.

The syntax is described below (note that **|** is a query separator): 
```
<query-syntax>              ::= <criterion> OR <criterion> "|" <query-syntax>
<criterion>                 ::= KEY [ <multivariate-criterion> OR <univariate-criterion> ]
<multivariate-criterion>    ::= <empty-list> OR <multivariate-operator> <multiple-values>
<empyt-list>                ::= "[]"
<multivariate-operator>     ::= "in" OR "notin"
<multiple-values>           ::= "[" <values> "]"
<values>                    ::= VALUE OR VALUE "|" <values>
<univariate-criterion>      ::= ["=" OR "!=" OR "eqornil" OR "lt" OR "gt"] VALUE

KEY is a sequence of characters with length from 1 to 255 characters, not containing a query separator.
VALUE is a sequence of characters with length from 1 to 255 characters.  
The query separator character (|) must be escaped with a backslash (\) if it is present.
Delimiter is exactly one whitespace: ' '
Example:
x in [val1|val2]|y = 5|z eqornil value with \| separator
```

## Operators

* Equals (**=**):
    - Works by testing whether the left operand's value and the right operand are equal
    - Example: `platform_id = my_platform_id`
* Not equals (**!=**):
    - Works by testing whether the left operand's value and the right operand are not equal
    - Example: `platform_id != my_platform_id`
* Equals or nil (**eqornil**)
    - Works by testing whether the left operand's value is equal to the right operand OR the left operand's value is NULL
    - Example: `platform_id eqornil my_platform_id`
* Greater than (**gt**):
    - Works by testing whether the left operand's value is greater than the right operand. Supports only numerical values.
    - Example: `id gt 5`
* Less than (**lt**)
    - Works by testing whether the left operand's value is less than the right operand. Supports only numerical values.
    - Example: `id lt 5`
* In (**in**)
    - Works by testing whether the left operand's value is contained in the right operand. Works only for list values of the right operand contained in square braces.
    - Example: `id in [5|6|7]`
* Not in (**notin**)
    - Works by testing whether the left operand's value is NOT contained in the right operand. Works only for list values of the right operand contained in square braces.
    - Example: `id notin [1|2|3]`

## Query Types

Queries let you select resources based on the value of either the resource fields or the labels attached to the resource (or both).

* Field Query  
A field query is a query that is performed on the fields of the object.  
Example: The `visibility` object has the field `platform_id` so one might say `Give me all visibilities for a platform with id X` which would translate to `/visibilities?fieldQuery=platform_id = X`

* Label Query  
A label query is a query that is performed on the labels associated with the object.
Example: You might label multiple visibilities with the label `test = true` saying that this is test data. So getting all non-test visibilities (these are the ones that either have `test = false` or they don't have a `test` label) would translate to `/visibilities?labelQuery=test eqornil false`

* Mixed Query  
A mixed query is a query that is performed both on fields and labels.  
Example: `Give me all non-test visibilities for platform with id X.` This would translate to `/visibilities?fieldQuery=platform_id = X&labelQuery=test eqornil false`

# Supported entities

Service Manager supports `field querying` for the following entities, where each entity might define which of its fields can be queried:
* visibility

Service Manager supports `label querying` for the following entities:
* visibility

# API

For description of the API see the specification
