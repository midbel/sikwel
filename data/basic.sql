select * from employees;

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


update employees set dept='IT' where manager=0;

delete from employees;

delete from employees where id = 89 RETURNING *;

INSERT INTO distributors (did, dname) VALUES 
	(7, 'Redline GmbH')
  ON CONFLICT (did) DO NOTHING;

INSERT INTO distributors (did, dname) VALUES 
	(5, 'Gizmo Transglobal'), 
	(6, 'Associated Computing, Inc')
  ON CONFLICT (did) DO UPDATE SET dname = EXCLUDED.dname;