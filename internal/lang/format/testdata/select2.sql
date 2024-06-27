select
	e.firstname,
	e.lastname,
	d.name
from employees e
join departments d on e.dept = d.id
where d.name in ('devel', 'helpdesk', 'infra')
	and d.active is not true
	and e.hired_date between '2024-01-01' and '2024-06-30'
order by e.lastname desc;