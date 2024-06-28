select
	firstname || ' ' || lastname fullname,
	title,
	published_year,
	editor
from books
join persons on p.id = b.author and p.id = b.editor
where published_year not between 2020 and 2024
	and topic not in ('it', 'devops', 'sql')
	and country in (select abbr from countries where area = 'europa')
order by published_year desc;

