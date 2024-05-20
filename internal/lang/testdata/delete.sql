delete from employees;
delete from employees where id <= 5 returning *;

truncate employees;
truncate table employees, departments;
truncate *;
truncate table employees continue identity cascade;