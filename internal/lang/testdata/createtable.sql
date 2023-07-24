create table employees (
	id    int not null primary key,
	name  varchar(12) not null,
	email varchar(64) unique,
	hired date check (hired_date >= current_date),
	dept  int not null,
	foreign key (dept) references departments(id),
	unique(name, email)
);