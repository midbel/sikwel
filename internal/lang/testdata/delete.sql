delete from employees;
delete from employees where id <= 5 returning *;