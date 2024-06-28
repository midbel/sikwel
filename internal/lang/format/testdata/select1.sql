select
	firstname,
	lastname
from employees
where exists(
	select
		name,
		active
	from departments
	where dept = 'it'
)
fetch first 100 rows only;