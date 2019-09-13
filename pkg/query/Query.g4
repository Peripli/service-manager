grammar Query ;
expression: criterion (Whitespace 'and' Whitespace expression)? EOF;
criterion: multivariate | univariate  ;
multivariate: Key Whitespace MultiOp Whitespace multiValues ;
univariate: Key Whitespace UniOp Whitespace Value ;
MultiOp: 'in' | 'notin' ;
multiValues: '(' manyValues? ')' ;
manyValues: Value (ValueSeparator manyValues)? ;
UniOp: 'eq' | 'ne' | 'gt' | 'lt' | 'ge' | 'le' | 'en' ;
Value: STRING | NUMBER | BOOLEAN | DATETIME ;
STRING: '\'' ('\\'. | '\'\'' | ~('\'' | '\\'))* '\'' ;
BOOLEAN: 'true' | 'false' ;
NUMBER : SIGN? DIGIT ('.' DIGIT)?;
SIGN: '-' | '+' ;
DATETIME: FOUR_DIGITS '-' TWO_DIGITS '-' TWO_DIGITS 'T' TWO_DIGITS ':' TWO_DIGITS ':' TWO_DIGITS '.' FIVE_DIGITS ('Z'|(SIGN TWO_DIGITS ':' TWO_DIGITS)) ;//2016-06-08T17:41:22Z
DIGIT: INTEGER+ ;
INTEGER: [0-9] ;
TWO_DIGITS: DIGIT DIGIT ;
FOUR_DIGITS: TWO_DIGITS TWO_DIGITS ;
FIVE_DIGITS: FOUR_DIGITS DIGIT ;
Key: [-_/a-zA-Z0-9\\]+ ;
ValueSeparator: ',' | ', ' ;
Whitespace: ' ' ;
WS : [ \t\r\n]+ -> skip ; // skip spaces, tabs, newlines