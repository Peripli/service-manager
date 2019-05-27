grammar Query ;
expression: criterion (Whitespace 'and' Whitespace expression)? EOF;
criterion: multivariate | univariate  ;
multivariate: Key Whitespace MultiOp Whitespace multiValues ;
univariate: Key Whitespace UniOp Whitespace Value ;
MultiOp: 'in' | 'notin' ;
multiValues: '(' manyValues? ')' ;
manyValues: Value (ValueSeparator manyValues)? ;
UniOp: 'eq' | 'neq' | 'gt' | 'lt' | 'gte' | 'lte' | 'eqornil' ;
Value: STRING | FLOAT | BOOLEAN ;
STRING: '\'' ('\\'. | '\'\'' | ~('\'' | '\\'))* '\'' ;
BOOLEAN: 'true' | 'false' ;
FLOAT : SIGN? [0-9]+ ('.' [0-9]+)?;
SIGN: '-' | '+' ;
Key: [-_/a-zA-Z0-9\\]+ ;
ValueSeparator: ',' | ', ' ;
Whitespace: ' ' ;
WS : [ \t\r\n]+ -> skip ; // skip spaces, tabs, newlines