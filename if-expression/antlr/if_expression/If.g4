grammar If;

program
   : statement+ EOF
   ;

statement
   : ifStatement
   ;

ifStatement
   : 'if' boolExpression
   ;

boolExpression
   : path #BoolExpressionPath
   | boolean #BoolExpressionBoolean
   | left=term '!=' right=term #BoolExpressionNotEqual
   | left=term '==' right=term #BoolExpressionEqual
   ;

parenBoolExpression
   : '(' boolExpression ')'
   ;

term
   : literal
   | parenBoolExpression
   ;

literal
   : nil
   | path
   | string
   | number
   | boolean
   ;

path
   : PATH
   ;

string
   : DOUBLE_STRING
   | SINGLE_STRING
   ;

number
   : DECIMAL_NUMBER
   ;

// TODO: Add a float type.

boolean
   : 'true'
   | 'false'
   ;

nil
   : 'nil'
   ;

DECIMAL_NUMBER
    : DecimalIntegerLiteral
    | DecimalIntegerLiteral '.' DecimalDigit*
    | '.' DecimalDigit+
    ;

DOUBLE_STRING
   : '"' (~[\r\n"])* '"'
   ;

SINGLE_STRING
   : '\'' (~[\r\n'])* '\''
   ;

PATH
   : ('.' [a-zA-Z0-9_-]*)+
   ;

STRING
   : [a-zA-Z0-9_]
   ;

WS
   : [ \r\n\t] -> skip
   ;

DOT
   : '.'
   ;

fragment DecimalDigit
   : [0-9]
   ;

fragment DecimalIntegerLiteral
 : '0'
 | [1-9] DecimalDigit*
 ;
