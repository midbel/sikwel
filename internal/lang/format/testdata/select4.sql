select
	firstname || ' ' || lastname fullname,
	name
from employees
join departments using (id_dept);