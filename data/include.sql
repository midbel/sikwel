select
concat_ws(e.firstname, e.lastname) as fullname,
case e.experiment
when 1 then 'senior'
when 2 then 'medior'
else 'junior'
end 'seniority'
from employees e;