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

select
*,
case
when e.hired > 10 then 'senior'
when e.hired > 5 and e.hired < 10 then 'medior'
else 'junior'
end 'seniority'
from employees e;

select
*,
case e.experiment
when 1 then 'senior'
when 2 then 'medior'
else 'junior'
end 'seniority'
from employees e;
