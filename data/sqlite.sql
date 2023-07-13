select e.firstname, e.lastname, e.dept, e.manager
from employees e
	left join (
		select e.id, e.firstname, e.lastname
		from employees e
		where manager is null
	) m on e.manager=m.id
order by dept collate nocase desc;

replace into employees(firstname, lastname) values
	('john', 'smith');

insert or replace into employees(firstname, lastname) values
	('john', 'smith');