TODOS

token
* ajout type placeholder: ?, :name, $1

rewrite rules
* rewrite binary expression like `VAL = NULL` as `VAL IS NULL`: ok
* rewrite `!=` operator as more standard `<>`: ok
* rewrite join using literal values as conditions using new subqueries
* replace fields of Writer UseCte and UseSubqueries by a new field which will contain all the rewrite rules: ok
* list of rewrite rules
  * binary expression
  * join condition
  * use standard operator
  * missing columns definition in cte
  * missing columns definition in create view statement

linting
* rule that check binary expression like `VAL = NULL` as warning
* rule that check used of non standard operator such as `!=`
* rule that check literal values in join conditions as warning
* rule that check that field from a subquery/cte exists in the query using it
* rule that check variables in a stored procedure

set properties to object (writer/parser/scanner/...) according to config file settings