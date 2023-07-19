IF age <> 0 THEN
	SELECT * FROM games g where g.age < 16;
END IF;