create table employees (
	id    int not null primary key,
	name  varchar(12) not null,
	email varchar(64) unique,
	hired date check (hired_date >= current_date),
	dept  int not null,
	salary numeric(6, 2) generated always as (salary *0.02) stored,
	foreign key (dept) references departments(id),
	constraint uniq_name_mail unique(name, email)
);