/* Повертає дані по заданому відео */
CREATE OR REPLACE FUNCTION public.return_video(
  IN  _idv character, /* id відео */
  OUT _title character, /* назва відео */
  OUT _description character,  /* опис відео */
  OUT _idch character, /* id плейлиста */
  OUT _chtitle character, /* назва каналу */
  OUT _publishedat timestamp with time zone,  /* Час публікації відео */
  OUT _count_metrics int /* Кількість метрик */,
  OUT _min_timemetric timestamp with time zone, /* максимальний час метрики */
  OUT _max_timemetric timestamp with time zone, /* мінімальний час метрики */ 
  OUT _duration bigint) /* тривалість відео */ AS
$BODY$

  DECLARE
    _id varchar;
    
  BEGIN
	SELECT id, title, TRIM(description), idch, publishedat, duration
	FROM video
	WHERE id = _idv 
	INTO _id, _title, _description, _idch, _publishedat, _duration;

	/* Перевірка чи є дані по відео*/
	IF _id IS NULL THEN
		RAISE EXCEPTION 'There is no video id: %', _idv USING HINT = 'Please check your video ID';
	ELSE
		SELECT COUNT(*), MAX(timemetric), MIN(timemetric) 
		FROM metric 
		WHERE idvideo = _idv 
		INTO _count_metrics, _max_timemetric, _min_timemetric;

		SELECT title FROM channel WHERE id = _idch INTO _chtitle;		

	END IF;		

	
  END;
$BODY$
  LANGUAGE plpgsql;
