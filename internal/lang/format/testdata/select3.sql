select
	d.name,
	count(*) total
from employees e
left join (
	select
		d.name
	from departments d
	where d.active is true
) d on e.dept = d.id
where d.name <> 'it'
group by d.name
order by d.name desc
limit 25;