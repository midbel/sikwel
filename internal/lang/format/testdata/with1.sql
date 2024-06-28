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
join departments on dept = id
where name collate "en_US"
order by name asc
limit 25 offset 10;