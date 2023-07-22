truncate table employees;
truncate table only employees *;
truncate table employees, departments;
truncate table employees cascade;
truncate table employees, departments * restart identity restrict;

select * from employees order by dept using <;

COPY employees TO STDOUT (DELIMITER '|');
COPY employees FROM STDIN (DELIMITER ',');
COPY departments(id, name) FROM '/tmp/departments.csv' WITH (FORMAT 'csv');
COPY employees TO PROGRAM 'gzip > /tmp/employees.csv.gz';
COPY (SELECT * FROM employees) TO '/tmp/employees.tmp';