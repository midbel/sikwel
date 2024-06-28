with
managers as (
	select
		salary
	from employees
	where manager is null
)
select
	firstname || ' ' || lastname,
	cast(salary as decimal(10, 3)),
	case
		when salary > 1000 then 'high'
		when salary <= 1000 and salary > 500 then 'medium'
		else 'low'
	end asset,
	cast(hired_date as smalldate),
	avg(salary) over (partition by dept order by salary, dept desc)
from employees
where not salary > all(select salary from managers)
	and salary >= (salary * 1.15)
	and dept = any(select name from departments where technic is true);
