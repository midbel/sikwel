with
managers as (
	select
		salary
	from employees
	where manager is null
)
select
	firstname || ' ' || lastname,
	salary
from employees
where not salary > all(select salary from managers)
	and dept = any(select name from departments where technic is true);
