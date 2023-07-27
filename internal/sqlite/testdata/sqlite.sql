begin transaction;
	select * from employees;
end;
begin immediate transaction;
	select * from employees;
rollback;
begin exclusive transaction;
	select * from employees;
commit;

delete from table where true;
delete from table where id=10;

insert or ignore into table as t(col1, col2) values ('value1', 'value2');
replace into table(col1, col2) values ('value1', 1);

insert into table as t(col) values (1)
	on conflict do update set col=0 where col=1;


select * from table limit 10;
select * from table limit 10 offset 5;
select * from table order by col collate NOCASE desc;
