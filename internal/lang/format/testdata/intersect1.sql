select
	e.firstname,
	e.lastname,
	d.name
from employees e
join departments d on e.dept = d.id
where e.hired_date >= '2024-01-01'
intersect all
select
	e.firstname,
	e.lastname,
	d.name
from employees e
join departments d on e.dept = d.id
where e.status <> 'junior';