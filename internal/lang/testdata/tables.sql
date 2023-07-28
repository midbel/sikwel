create table employees (
	id    int not null primary key,
	name  varchar(12) not null,
	email varchar(64) unique,
	hired date check (hired_date >= current_date),
	dept  int not null default 'support',
	salary numeric(6, 2) generated always as (salary *0.02) stored,
	foreign key (dept) references departments(id),
	constraint uniq_name_mail unique(name, email)
);

create table if not exists departments (
	id serial not null,
	name varchar(12) not null,
	primary key (id),
	constraint unique_name unique(name)
);

alter table employees rename to people;
alter table employees rename name to fullname;
alter table employees rename column name to fullname;
alter table employees drop hired;
alter table employees drop column hired;