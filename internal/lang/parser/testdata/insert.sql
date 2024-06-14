insert into employees (id, firstname, lastname, manager) values
	(default, 'john', 'smith', null),
	(default, 'john', 'brown', 1)
returning id, manager;

insert into employees (id, firstname, lastname, manager) values
	(default, 'john', 'smith', null)
on conflict do nothing;

insert into departments(name) select dept from employees;

insert into employes (id, name, dept) values
	(default, 'john smith', 'it')
	on conflict do update set name='john smith 2'
	where id=1;