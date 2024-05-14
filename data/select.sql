SELECT 
    kind, 
    sum(len) AS total
FROM films
GROUP BY kind
HAVING sum(len) < interval '5 hours';

SELECT 
    f.title, 
    f.did, 
    d.name, 
    f.date_prod, 
    f.kind
FROM distributors d 
    JOIN films f USING (did);

SELECT distributors.name
FROM distributors
WHERE distributors.name LIKE 'W%'
UNION ALL
SELECT actors.name
FROM actors
WHERE actors.name LIKE 'W%';