truncate table employees;
truncate table only employees *;
truncate table employees, departments;
truncate table employees cascade;
truncate table employees, departments * restart identity restrict;

select * from employees order by dept using <;