merge into employees e using persons p on e.id=p.id
when matched then
	update set dept='it'
when matched and dept is null then
	delete
when not matched then
	insert (firstname, lastname, dept) values('foo', 'bar', 'it');