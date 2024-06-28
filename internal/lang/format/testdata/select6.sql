with
managers as (
	select
		salary
	from employees
	where manager is null
)
select
	firstname || ' ' || lastname,
	salary,
	avg(salary) over (partition by dept order by salary, dept desc)
from employees
where not salary > all(select salary from managers)
	and salary >= (salary * 1.15)
	and dept = any(select name from departments where technic is true);
