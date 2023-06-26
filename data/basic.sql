select * from movies;

select m.title, d.name, m.kind, m.year 
from movies m 
join distributors d on m.distrib_id=d.id;

select m.title, d.name, m.kind, m.year 
from movies m 
join distributors d using (distrib);

select m.title, m.kind, m.year, a.name
from actors a 
join movies_actors ma on a.id=ma.actor
join movies m on m.id=ma.movie
where a.name like 'w%' and age >= 18 and m.duration >= 90;

select upper(a.name) as actor, ifnull(count(m.*), 0)
from actors a 
join movies_actors ma on a.id=ma.actor
join movies m on m.id=ma.movie
group by a.name
order by actor desc
limit 10, 20;