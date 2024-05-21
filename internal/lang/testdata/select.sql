-- all from employees
select * from employees;
select * from db.employees e;
select * from employees limit 10;
select * from employees limit 10 offset 5;
select * from employees offset 5 rows fetch next 5 rows only;

select * from employees where not exists(select 1 from employees where dept='it');
select dept, count(*) as total from employees group by dept having total != 0;
select 
name, case when age > 40 then 'senior' else 'junior' end seniority 
from employees;

select "first" || ' ' || "last" as "full" from employees;
select  name from employees order by salary desc;
select concat_ws(' ', firstname, lastname) 
from employees  
where salary >= 1000 and dept='it';
select e.dept, count(e.id) 
from employees e 
where e.salary >= 1000 and e.manager is null group by e.dept;
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

select row_number() over (partition by hired_date order by year) as ronum, * from employees;
select * from table window win as (partition by field order by other);

select * from row(1, 'john smith', 'it', '2023-07-21');

select count(*) from employees where dept='it'
union all
select count(*) from employees where dept='comm';

select * from employees where dept in ('it', 'hr');
select * from employees where dept in (select name from departments where active is not null);
select * from employees where hired_date between '2024-05-01' and '2024-05-30';
select * from employees where exists (select 1 from employees where dept like '%it%');

select * from employees where dept notnull;
select * from employees where dept isnull;

@include 'values.sql';