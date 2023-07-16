select * from employees;
select * from employees limit 10;
select * from employees limit 10 offset 5;
select * from employees offset 5 rows fetch next 5 rows only;
select 
firstname, lastname 
from employees 
order by salary desc;
select
concat_ws(' ', firstname, lastname)
from employees 
where salary >= 1000 and dept='it';
select
e.dept
count(e.id)
from employees e
where e.salary >= 1000 and e.manager is null
group by e.dept;
with managers as (
	select
	id,
	concat_ws(' ', firstname, lastname)
	from employees
	where manager is null
)
select * from employees join managers using(id);
select * from employees e join managers m on e.manager=m.id;
select *
from employees e 
join (
	select
	id,
	concat_ws(' ', firstname, lastname)
	from employees
	where manager is null
) m on e.manager=m.id;