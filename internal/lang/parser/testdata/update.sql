update employees set dept = 'it' where id >= 10;

update employees set (name,dept,salary)=('john smith', 'it', 100.0) where id=15;

update employees set 
	(dept,salary)=(select dept, salary from employees where id=10),
	working_hours=working_hours+10
where id=17
returning working_hours;

update departments set
	working_hours=DEFAULT,
	contact=e.manager
from employees e where e.manager is not null and e.dept = 'it';
