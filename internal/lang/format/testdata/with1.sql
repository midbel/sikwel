with
departments(name, active) as (
	select
		name,
		state
	from departments
	where name is not null
		and state is true
)
select
	firstname,
	lastname,
	name
from employees
join departments on dept=id
order by name
limit 25 offset 10;