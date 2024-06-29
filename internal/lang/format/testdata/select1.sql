@format as true;
@format keepspace false;
@format quote true;
@format upperize keyword;

select
	e.firstname fname,
	e.lastname lname
from employees e
where exists(
	select
		d.name,
		d.active
	from departments d
	where d.dept = 'it'
)
fetch first 100 rows only;
--
SELECT
	"e"."firstname" AS "fname",
	"e"."lastname" AS "lname"
FROM "employees" AS "e"
WHERE EXISTS(
	SELECT
		"d"."name",
		"d"."active"
	FROM "departments" AS "d"
	WHERE "d"."dept"='it'
)
FETCH FIRST 100 ROWS ONLY;