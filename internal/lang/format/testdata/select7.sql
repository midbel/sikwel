@format as true;
@format rewrite "subquery-as-cte";
select
	e.firstname,
	e.lastname
from employees e
join (
	select id, name 
	from departments 
	where technic is true and active is true
) d on e.dept=d.id;
--
with
d(id, name) as (
	select
		id,
		name
	from departments
	where technic is true
		and active is true
)
select
	e.firstname,
	e.lastname
from employees as e
join d on e.dept = d.id;