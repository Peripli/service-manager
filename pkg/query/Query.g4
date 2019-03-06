grammar Query ;
expression: criterion ('and' expression)? EOF;
criterion: multivariate | univariate ;
multivariate: Key MultiOp multiValues ;
univariate: Key UniOp Value ;
MultiOp: 'in' | 'notin' ;
multiValues: '[' manyValues? ']' ;
manyValues: Value (ValueSeparator manyValues)? ;
UniOp: 'eq' | 'neq' | 'eqornil' | 'gt' | 'lt' ;
Key: [-_/a-zA-Z0-9\\]+ ;
Value: '\'' ('\\'. | '\'\'' | ~('\'' | '\\'))* '\'' ;
ValueSeparator: ',' | ', ' ;
WS : [ \t\r\n]+ -> skip ; // skip spaces, tabs, newlines