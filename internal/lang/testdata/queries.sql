-- all from employees
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
-- join with select
select *
from employees e 
join (
	select
	id,
	concat_ws(' ', firstname, lastname)
	from employees
	where manager is null
) m on e.manager=m.id;

-- insert statements
insert into employees (id, firstname, lastname, manager) values
	(default, 'john', 'smith', null),
	(default, 'john', 'brown', 1)
returning id, manager;

insert into employees (id, firstname, lastname, manager) values
	(default, 'john', 'smith', null)
on conflict do nothing;

insert into departments(name) select dept from employees;

update employees set dept = 'it' where id >= 10;

delete from employees;

delete from employees where id <= 5 returning *;