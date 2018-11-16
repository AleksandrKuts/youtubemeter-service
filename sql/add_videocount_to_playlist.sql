/* Додає  */
CREATE OR REPLACE FUNCTION change_video() RETURNS TRIGGER AS 
$BODY$
BEGIN
	raise notice '%', TG_OP;
	IF (TG_OP = 'INSERT') THEN
		UPDATE playlist SET countvideo = countvideo + 1	WHERE id = NEW.idpl;
		RETURN NEW;
	ELSIF (TG_OP = 'DELETE') THEN
		UPDATE playlist SET countvideo = countvideo - 1	WHERE id = OLD.idpl;
		RETURN OLD;
	END IF;
END
$BODY$ LANGUAGE PLPGSQL;

CREATE TRIGGER tr_change_video
AFTER INSERT OR DELETE ON video
    FOR EACH ROW EXECUTE PROCEDURE change_video();
