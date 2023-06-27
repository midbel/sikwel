select * from employees;

select d.name, count(e.id) members 
from departments d
left outer join employees e on d.id=e.dept
group by d.name
having members > 10
order by members desc;

select
e.*,
m.manager,
d.name
from employees e
join departments d on e.dept=d.id
join (select id, concat_ws('firstname', 'lastname') manager from employees) m on e.manager=m.id;