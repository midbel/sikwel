CREATE OR REPLACE PROCEDURE TEST(
	IN x INT,
	OUT y REAL DEFAULT 0, 
	INOUT msg VARCHAR(12) = 'test'
) LANGUAGE SQL
BEGIN
	DECLARE total INT;
	SET total = x * total;
	IF total <> 0 THEN
		SET msg = 'not zero';
	END IF;
	SELECT * FROM employes;
	RETURN 0;
END;