-- all from employees
select * from employees;
select * from db.employees e;
select * from employees limit 10;
select * from employees limit 10 offset 5;
select * from employees offset 5 rows fetch next 5 rows only;

select  name from employees order by salary desc;
select concat_ws(' ', firstname, lastname) from employees  where salary >= 1000 and dept='it';
select e.dept count(e.id) from employees e where e.salary >= 1000 and e.manager is null group by e.dept;
-- test with CTE
with managers as (
	select
	id,
	concat_ws(' ', firstname, lastname)
	from employees
	where manager is null
)
select * from employees join managers using(id);

select * from employees e join managers m on e.manager=m.id;

select * from employees e 
join (
	select
	id,
	concat_ws(' ', firstname, lastname)
	from employees
	where manager is null
) m on e.manager=m.id;

select cast(hired_date as int(32)) from employees;
select max(hired_date) filter(where hired_date > 2023) from employees;

select * from table window win as (partition by field order by other);