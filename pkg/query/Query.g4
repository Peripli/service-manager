grammar Query ;

expression: criterions EOF ;
criterions: criterion (Concat criterions)? ;
criterion: multivariate | univariate  ;
multivariate: Key Whitespace MultiOp Whitespace multiValues ;
univariate: Key Whitespace UniOp Whitespace Value ;
multiValues: OpenBracket manyValues? CloseBracket ;
manyValues: Value (ValueSeparator manyValues)? ;

MultiOp:  'in' | 'notin' ;
UniOp: 'eq' | 'ne' | 'gt' | 'lt' | 'ge' | 'le' | 'en' ;
Concat: Whitespace 'and' Whitespace ;
Value: STRING | NUMBER | BOOLEAN | DATETIME ;
ValueSeparator: ',' | ', ' ;
Key: [-_/a-zA-Z0-9\\]+ ;
OpenBracket: '(' ;
CloseBracket: ')';

fragment BOOLEAN: 'true' | 'false' ;
fragment STRING: '\'' ('\\'. | '\'\'' | ~('\'' | '\\'))* '\'' ;
fragment YEAR : DIGIT DIGIT DIGIT DIGIT ;
fragment MONTH : DIGIT DIGIT ;
fragment DAY : DIGIT DIGIT ;
fragment DELIM : 'T' | 't' ;
fragment HOUR : DIGIT DIGIT ;
fragment MINUTE : DIGIT DIGIT ;
fragment SECOND : DIGIT DIGIT ;
fragment SECFRAC : '.' DIGIT+ ;
fragment NUMOFFSET : ('+' | '-') HOUR ':' MINUTE ;
fragment OFFSET : 'Z' | NUMOFFSET ;
fragment PARTIAL_TIME : HOUR ':' MINUTE ':' SECOND SECFRAC? ;
fragment FULL_DATE : YEAR '-' MONTH '-' DAY ;
fragment FULL_TIME : PARTIAL_TIME OFFSET ;
fragment DATETIME : FULL_DATE DELIM FULL_TIME ;
fragment FIVE_DIGITS: FOUR_DIGITS DIGIT ;
fragment FOUR_DIGITS: TWO_DIGITS TWO_DIGITS ;
fragment TWO_DIGITS: DIGIT DIGIT ;
fragment NUMBER : SIGN? DIGIT ('.' DIGIT)?;
fragment SIGN: '-' | '+' ;
fragment DIGIT: INTEGER+ ;
fragment INTEGER: [0-9] ;
Whitespace: ' ' ;

WS : [ \t\r\n]+ -> skip ; // skip spaces, tabs, newlines