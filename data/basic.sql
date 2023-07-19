select * from employees;

select * from employees where manager is null;
select * from employees where manager is not null;

select d.name, count(e.id) members 
from departments d
left outer join employees e on d.id=e.dept
group by d.name
having members > 10
order by members desc;

select
e.*,
m.manager,
d.name
from employees e
join departments d on e.dept=d.id
join (select id, concat_ws('firstname', 'lastname') manager from employees) m on e.manager=m.id;

select id, "first" || ' ' || "last" from employees;

select
*,
case
when e.hired > 10 then 'senior'
when e.hired > 5 and e.hired < 10 then 'medior'
else 'junior'
end 'seniority'
from employees e;

@include 'include.sql';

insert into employees(firstname, lastname) select * from users;
insert into employees(firstname, lastname) values
	('john', 'smith'),
	('warren', 'dennis');

insert into employees(firstname, lastname)
	values ('john', 'smith')
	on conflict do nothing
	returning *;

insert into employes (name, dept) values 
	('john smith', 'it')
  on conflict (dept) do nothing;

insert into employes(id, name, dept) values
	(default, 'john smith', 'it')
	on conflict (dept) do update set dept='other'
	where manager = 0
	returning *;

update employees set dept='it' where manager=0;

delete from employees;

delete from employees where id = 89 returning *;

with managers(name, dept) as (
	select name, dept from employees where manager is null
), departments as (
	select name from departments
)
select * from managers m join departments d on m.dept=d.id;

-- declare i int default 10;
-- declare f real;
-- declare v varchar(32);
-- declare c char(1);

select cast(hired_date as int) from employees;

select
max(hired_date) filter(where hired_date > 0) 
from employees;

select 
name,
row_number() over(partition by year(hired_date))
from employees;

values ROW(1, 2, 3);