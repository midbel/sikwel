TODOS

token
* ajout type placeholder: ?, :name, $1

rewrite rules
* rewriting subquery as cte when subquery as no alias, reuse the name of the main table as alias by taking from the query
* rewrite join using literal values as conditions using new subqueries

linting
* rule that check binary expression like `VAL = NULL` as warning
* rule that check used of non standard operator such as `!=`
* rule that check literal values in join conditions as warning
* rule that check that field from a subquery/cte exists in the query using it
* rule that check variables in a stored procedure

set properties to object (writer/parser/scanner/...) according to config file settings