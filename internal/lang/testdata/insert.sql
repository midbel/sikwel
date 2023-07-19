insert into employees (id, firstname, lastname, manager) values
	(default, 'john', 'smith', null),
	(default, 'john', 'brown', 1)
returning id, manager;

insert into employees (id, firstname, lastname, manager) values
	(default, 'john', 'smith', null)
on conflict do nothing;

insert into departments(name) select dept from employees;