/* Повертає дані по заданому відео */
CREATE OR REPLACE FUNCTION public.return_video(
  IN  _idv character, /* id відео */
  OUT _idpl character, /* id плейлиста */
  OUT _title character, /* назва відео */
  OUT _description character,  /* опис відео */
  OUT _chtitle character, /* назва каналу */
  OUT _chid character, /* id каналу */
  OUT _publishedat timestamp with time zone,  /* Час публікації відео */
  OUT _count_metrics int /* Кількість метрик */,
  OUT _min_timemetric timestamp with time zone, /* максимальний час метрики */
  OUT _max_timemetric timestamp with time zone) /* мінімальний час метрики */ AS
$BODY$

  DECLARE
    _id varchar;
    _RET_NOT_FOUND character := "не знайдено"; 
    
  BEGIN
	SELECT id, idpl, TRIM(title), TRIM(description), TRIM(chtitle), chid, publishedat FROM video 
	WHERE id = _idv INTO _id, _idpl, _title, _description, _chtitle, _chid, _publishedat;

	/* Перевірка чи є дані по відео*/
	IF _id IS NULL THEN
		_idpl = _RET_NOT_FOUND;
		_title = _RET_NOT_FOUND;
		_description = _RET_NOT_FOUND;
		_chtitle = _RET_NOT_FOUND;
		_chid = _RET_NOT_FOUND;
		_publishedat = now();
		_count_metrics = 0;
		_max_timemetric = now();
		_min_timemetric = now();
	ELSE
		SELECT COUNT(*), MAX(timemetric), MIN(timemetric) FROM metric 
		WHERE idvideo = _idv INTO _count_metrics, _max_timemetric, _min_timemetric;		
	END IF;		

	
  END;
$BODY$
  LANGUAGE plpgsql;