select
	d.name,
	(
		select
			count(*)
		from employees
		where dept = d.id
	) total,
	(
		select
			count(*)
		from employees
		where dept <> d.id
	)
from departments d
order by d.name asc;