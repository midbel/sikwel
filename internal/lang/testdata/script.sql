IF age <> 0 THEN
	SELECT * FROM games g where g.age < 16;
END IF;

WHILE TRUE DO
	SELECT * FROM employees;
END WHILE;

DECLARE TOTAL INT DEFAULT 10;
DECLARE MESSAGE VARCHAR(64);

SET TOTAL = 0;

BEGIN
	SELECT * FROM employees;
	DELETE FROM departments;
END;

IF TOTAL < 10 THEN
	SET MESSAGE = 'lesser';
	SET TOTAL = 0;
ELSIF TOTAL > 10 THEN
	SET MESSAGE = 'greater';
	SET TOTAL = 1;
ELSE
	SET MESSAGE = 'equal';
END IF;

@include 'transactions.sql';