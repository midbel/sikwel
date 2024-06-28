@format as true;
@format keepspace false;

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
select
	e.firstname as fname,
	e.lastname as lname
from employees as e
where exists(
	select
		d.name,
		d.active
	from departments as d
	where d.dept='it'
)
fetch first 100 rows only;