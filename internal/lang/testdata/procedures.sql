CREATE OR REPLACE PROCEDURE TEST(
	IN x INT,
	OUT y REAL DEFAULT 0, 
	INOUT msg VARCHAR(12) = 'test'
) 
LANGUAGE SQL
BEGIN
	DECLARE total INT;
	SET total = x * total;
	IF total <> 0 THEN
		SET msg = 'not zero';
	ELSIF total = 1 THEN
		SET msg = 'one';
		VALUES current_date();
	ELSE
		SET msg = 'zero';
	END IF;
	SELECT * FROM employes;
	RETURN 0;
END;