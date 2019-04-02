

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;


CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;



COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';



CREATE FUNCTION public.change_video() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
IF (TG_OP = 'INSERT') THEN
UPDATE channel SET countvideo = countvideo + 1 WHERE id = NEW.idch;
RETURN NEW;
ELSIF (TG_OP = 'DELETE') THEN
UPDATE channel SET countvideo = countvideo - 1 WHERE id = OLD.idch;
RETURN OLD;
END IF;
END
$$;



CREATE FUNCTION public.return_metrics(_idv character) RETURNS TABLE(commentcount bigint, likecount bigint, dislikecount bigint, viewcount bigint, timemetric timestamp with time zone)
    LANGUAGE plpgsql
    AS $$
  DECLARE _MAX_RETURN_COUNT_ROWS CONSTANT int := 100; /* максимальна кількість повертаних рядків (може бути більше на перший та останній) */
  DECLARE _count_metrics bigint;
  DECLARE _min_timemetric timestamp with time zone;
  DECLARE _max_timemetric timestamp with time zone;
  DECLARE _step_index float;
  DECLARE _indexes bigint[] ;

  BEGIN
	/* Отримуємо кількість записів які задовольняють запиту - це необхідно для подальших розрахунків, 
	та ознаку останнього запису - його додаємо обов'язково */
	SELECT COUNT(*), MIN(m.timemetric), MAX(m.timemetric) 
	  FROM metric m 
	  WHERE m.idvideo = _idv 
	  INTO _count_metrics, _min_timemetric, _max_timemetric;

	/* Перевірка чи є дані по відео*/
	IF _count_metrics = 0 THEN
		RAISE EXCEPTION 'There are no metrics for this video id: %', _idv USING HINT = 'Please check your video ID';
	END IF;	

	/* Число записів менше максимально заданого, тому повертаємо всі */
	IF _MAX_RETURN_COUNT_ROWS > _count_metrics THEN
		RETURN QUERY 
		SELECT m.commentcount, m.likecount, m.dislikecount, m.viewcount, m.timemetric 
		  FROM  metric m
		  WHERE m.idvideo = _idv 
		  ORDER BY timemetric;
		
	/* Число записів більше максимально заданого, тому повертаємо точну кількість розподілену по інтервалу. 
	Для цього використовуємо номери записів */
	ELSE
		/* Визначаємо номера записів які вибираються із загального інтервалу */
		_step_index := _count_metrics::float / _MAX_RETURN_COUNT_ROWS;
		FOR i IN 0.._MAX_RETURN_COUNT_ROWS - 1
		LOOP
			_indexes[i] := round(i * _step_index);
		END LOOP;	

		RETURN QUERY
		SELECT m.commentcount, m.likecount, m.dislikecount, m.viewcount, m.timemetric FROM 
		  (
			/* Цей підзапит потрібен щоб додати колонку с номером запису для подальшої фільтрації */
			SELECT ROW_NUMBER() OVER () as rnum, *
			  FROM metric s 
			  /* Вибираємо дані по відео id, даних буде більше чим максимально задано параметром: _MAX_RETURN_COUNT_ROWS */
			  WHERE s.idvideo = _idv 
			  ORDER BY s.timemetric
		  ) m 
		  /* фільтруємо записи по їх номеру: додаємо тільки обрані номери записів та останній запис,
		  тепер записів буде не більше чим максимально задано параметром: _MAX_RETURN_COUNT_ROWS (плюс останній) */
		  WHERE m.rnum = ANY(_indexes) 
		     OR m.timemetric = _min_timemetric 
		     OR m.timemetric = _max_timemetric;
	END IF;
      	
  END;
$$;



CREATE FUNCTION public.return_metrics(_idv character, _from_ch character, _to_ch character) RETURNS TABLE(commentcount bigint, likecount bigint, dislikecount bigint, viewcount bigint, timemetric timestamp with time zone)
    LANGUAGE plpgsql
    AS $$
  DECLARE _MAX_RETURN_COUNT_ROWS CONSTANT int := 100; /* максимальна кількість повертаних рядків (може бути більше на перший та останній) */
  DECLARE _count_metrics bigint;
  DECLARE _min_timemetric timestamp;
  DECLARE _max_timemetric timestamp with time zone;
  DECLARE _step_index float;
  DECLARE _indexes bigint[] ;

  DECLARE _from timestamp with time zone = '-infinity'::timestamp with time zone;
  DECLARE _to timestamp with time zone = 'infinity'::timestamp with time zone;
  
  BEGIN

	IF _from_ch != '' THEN
		_from := _from_ch::timestamp with time zone;
	END IF;	
	IF _to_ch != '' THEN
		_to := _to_ch::timestamp with time zone;
	END IF;	

	RAISE NOTICE 'sss % %', _from, _to;


	/* Отримуємо кількість записів які задовольняють запиту - це необхідно для подальших розрахунків, 
	та ознаку останнього запису - його додаємо обов'язково */
	SELECT COUNT(*), MIN(m.timemetric), MAX(m.timemetric) 
	  FROM metric m 
	  WHERE m.idvideo = _idv 
	    AND m.timemetric >= _from::timestamp with time zone 	
	    AND m.timemetric <= _to::timestamp with time zone  
	  INTO _count_metrics, _min_timemetric, _max_timemetric;

	/* Перевірка чи є дані по відео*/
	IF _count_metrics = 0 THEN
		RAISE EXCEPTION 'There are no metrics for this video id: %', _idv USING HINT = 'Please check your video ID';
	END IF;	

	/* Число записів менше максимально заданого, тому повертаємо всі */
	IF _MAX_RETURN_COUNT_ROWS > _count_metrics THEN
		RETURN QUERY 
		SELECT m.commentcount, m.likecount, m.dislikecount, m.viewcount, m.timemetric 
		  FROM  metric m
		  WHERE m.idvideo = _idv 
		    AND m.timemetric >= _from::timestamp with time zone 
		    AND m.timemetric <= _to::timestamp with time zone 
		  ORDER BY timemetric;

	/* Число записів більше максимально заданого, тому повертаємо точну кількість розподілену по інтервалу. 
	Для цього використовуємо номери записів */
	ELSE
		/* Визначаємо номера записів які вибираються із загального інтервалу */
		_step_index := _count_metrics::float / _MAX_RETURN_COUNT_ROWS;
		FOR i IN 0.._MAX_RETURN_COUNT_ROWS - 1
		LOOP
			_indexes[i] := round(i * _step_index);
		END LOOP;	

		RETURN QUERY
		SELECT m.commentcount, m.likecount, m.dislikecount, m.viewcount, m.timemetric from 
		  (
			/* Цей підзапит потрібен щоб додати колонку с номером запису для подальшої фільтрації */
			SELECT ROW_NUMBER() OVER () as rnum, *
			  FROM metric s 
			  /* Вибираємо дані по відео id, даних буде більше чим максимально задано параметром: _MAX_RETURN_COUNT_ROWS */
			  WHERE s.idvideo = _idv 
			    AND s.timemetric >= _from::timestamp with time zone 
			    AND s.timemetric <= _to::timestamp with time zone 
			  ORDER BY s.timemetric
		  ) m 
		  /* фільтруємо записи по їх номеру: додаємо тільки обрані номери записів та останній запис,
		  тепер записів буде не більше чим максимально задано параметром: _MAX_RETURN_COUNT_ROWS (плюс останній) */
		  WHERE m.rnum = ANY(_indexes) 
		     OR m.timemetric = _min_timemetric 
		     OR m.timemetric = _max_timemetric;

	END IF;
      	
  END;
$$;



CREATE FUNCTION public.return_video(_idv character, OUT _title character, OUT _description character, OUT _idch character, OUT _chtitle character, OUT _publishedat timestamp with time zone, OUT _count_metrics integer, OUT _min_timemetric timestamp with time zone, OUT _max_timemetric timestamp with time zone) RETURNS record
    LANGUAGE plpgsql
    AS $$

  DECLARE
    _id varchar;
    
  BEGIN
	RAISE NOTICE '1 (%, %)', _idv, _id;
	SELECT id, title, TRIM(description), idch, publishedat
	FROM video
	WHERE id = _idv 
	INTO _id, _title, _description, _idch, _publishedat;
	RAISE NOTICE '2 (%, %)', _idv, _id;

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
$$;


SET default_with_oids = false;


CREATE TABLE public.channel (
    id character(24) NOT NULL,
    enable boolean,
    title character(80),
    timeadd timestamp with time zone DEFAULT now(),
    countvideo integer DEFAULT 0
);



CREATE TABLE public.metric (
    id integer NOT NULL,
    idvideo character(11) NOT NULL,
    commentcount bigint DEFAULT 0,
    dislikecount bigint DEFAULT 0,
    likecount bigint DEFAULT 0,
    viewcount bigint DEFAULT 0,
    timemetric timestamp with time zone
);



CREATE SEQUENCE public.metrics_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;



ALTER SEQUENCE public.metrics_id_seq OWNED BY public.metric.id;



CREATE TABLE public.video (
    id character(11) NOT NULL,
    idch character(24) NOT NULL,
    title character(100) DEFAULT ''::bpchar,
    description character varying(5000),
    publishedat timestamp with time zone,
    duration bigint
);



ALTER TABLE ONLY public.metric ALTER COLUMN id SET DEFAULT nextval('public.metrics_id_seq'::regclass);



ALTER TABLE ONLY public.channel
    ADD CONSTRAINT channel_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.metric
    ADD CONSTRAINT metrics_pkey PRIMARY KEY (id);



ALTER TABLE ONLY public.video
    ADD CONSTRAINT video_pkey PRIMARY KEY (id);



CREATE INDEX metric_idvideo_timemetric_idx ON public.metric USING btree (idvideo, timemetric);



CREATE INDEX video_idch_idx ON public.video USING btree (idch);



CREATE TRIGGER tr_change_video AFTER INSERT OR DELETE ON public.video FOR EACH ROW EXECUTE PROCEDURE public.change_video();



ALTER TABLE ONLY public.metric
    ADD CONSTRAINT metrics_idv_fkey FOREIGN KEY (idvideo) REFERENCES public.video(id);



ALTER TABLE ONLY public.video
    ADD CONSTRAINT video_idch_fkey FOREIGN KEY (idch) REFERENCES public.channel(id);



