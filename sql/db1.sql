--
-- PostgreSQL database dump
--

-- Dumped from database version 9.5.14
-- Dumped by pg_dump version 9.5.14

-- Started on 2018-11-14 16:45:09 EET

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

DROP DATABASE youtube_statistics;
--
-- TOC entry 2172 (class 1262 OID 121644)
-- Name: youtube_statistics; Type: DATABASE; Schema: -; Owner: youtube
--

CREATE DATABASE youtube_statistics WITH TEMPLATE = template0 ENCODING = 'UTF8' LC_COLLATE = 'ru_UA.UTF-8' LC_CTYPE = 'ru_UA.UTF-8';


ALTER DATABASE youtube_statistics OWNER TO youtube;

\connect youtube_statistics

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 1 (class 3079 OID 12397)
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: 
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- TOC entry 2175 (class 0 OID 0)
-- Dependencies: 1
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


--
-- TOC entry 198 (class 1255 OID 121645)
-- Name: return_metrics(character); Type: FUNCTION; Schema: public; Owner: youtube
--

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


ALTER FUNCTION public.return_metrics(_idv character) OWNER TO youtube;

--
-- TOC entry 199 (class 1255 OID 121646)
-- Name: return_metrics(character, character, character); Type: FUNCTION; Schema: public; Owner: youtube
--

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


ALTER FUNCTION public.return_metrics(_idv character, _from_ch character, _to_ch character) OWNER TO youtube;

--
-- TOC entry 197 (class 1255 OID 121647)
-- Name: return_video(character); Type: FUNCTION; Schema: public; Owner: youtube
--

CREATE FUNCTION public.return_video(_idv character, OUT _idch character, OUT _title character, OUT _description character, OUT _chtitle character, OUT _publishedat timestamp with time zone, OUT _count_metrics integer, OUT _min_timemetric timestamp with time zone, OUT _max_timemetric timestamp with time zone) RETURNS record
    LANGUAGE plpgsql
    AS $$

  DECLARE
    _id varchar;
  BEGIN
	SELECT id, chid, TRIM(title), TRIM(description), TRIM(chtitle), publishedat FROM video 
	WHERE id = _idv INTO _id, _chid, _title, _description, _chtitle, _publishedat;

	/* Перевірка чи є дані по відео*/
	IF _id IS NULL THEN
		RAISE EXCEPTION 'There are no video id: %', _idv USING HINT = 'Please check your video ID';
	ELSE
		SELECT COUNT(*), MAX(timemetric), MIN(timemetric) FROM metric 
		WHERE idvideo = _idv INTO _count_metrics, _max_timemetric, _min_timemetric;		
	END IF;		

	
  END;
$$;


ALTER FUNCTION public.return_video(_idv character, OUT _chid character, OUT _title character, OUT _description character, OUT _chtitle character, OUT _publishedat timestamp with time zone, OUT _count_metrics integer, OUT _min_timemetric timestamp with time zone, OUT _max_timemetric timestamp with time zone) OWNER TO youtube;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- TOC entry 181 (class 1259 OID 121648)
-- Name: metric; Type: TABLE; Schema: public; Owner: youtube
--

CREATE TABLE public.metric (
    id integer NOT NULL,
    idvideo character(11) NOT NULL,
    commentcount bigint DEFAULT 0,
    dislikecount bigint DEFAULT 0,
    likecount bigint DEFAULT 0,
    viewcount bigint DEFAULT 0,
    timemetric timestamp with time zone
);


ALTER TABLE public.metric OWNER TO youtube;

--
-- TOC entry 182 (class 1259 OID 121655)
-- Name: metrics_id_seq; Type: SEQUENCE; Schema: public; Owner: youtube
--

CREATE SEQUENCE public.metrics_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.metrics_id_seq OWNER TO youtube;

--
-- TOC entry 2180 (class 0 OID 0)
-- Dependencies: 182
-- Name: metrics_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: youtube
--

ALTER SEQUENCE public.metrics_id_seq OWNED BY public.metric.id;


--
-- TOC entry 183 (class 1259 OID 121657)
-- Name: playlist; Type: TABLE; Schema: public; Owner: youtube
--

CREATE TABLE public.channel (
    id character(24) NOT NULL,
    enable boolean,
    title character(80),
    timeadd timestamp with time zone DEFAULT now()
);


ALTER TABLE public.channel OWNER TO youtube;

--
-- TOC entry 184 (class 1259 OID 121660)
-- Name: video; Type: TABLE; Schema: public; Owner: youtube
--

CREATE TABLE public.video (
    id character(11) NOT NULL,
    idch character(24) NOT NULL,
    title character(100) DEFAULT ''::bpchar,
    description character varying(5000),
    chtitle character(100) DEFAULT ''::bpchar,
    publishedat timestamp with time zone
);


ALTER TABLE public.video OWNER TO youtube;

--
-- TOC entry 2039 (class 2604 OID 121669)
-- Name: id; Type: DEFAULT; Schema: public; Owner: youtube
--

ALTER TABLE ONLY public.metric ALTER COLUMN id SET DEFAULT nextval('public.metrics_id_seq'::regclass);


--
-- TOC entry 2045 (class 2606 OID 121671)
-- Name: metrics_pkey; Type: CONSTRAINT; Schema: public; Owner: youtube
--

ALTER TABLE ONLY public.metric
    ADD CONSTRAINT metrics_pkey PRIMARY KEY (id);


--
-- TOC entry 2047 (class 2606 OID 121673)
-- Name: playlist_pkey; Type: CONSTRAINT; Schema: public; Owner: youtube
--

ALTER TABLE ONLY public.channel
    ADD CONSTRAINT channel_pkey PRIMARY KEY (id);


--
-- TOC entry 2050 (class 2606 OID 121675)
-- Name: video_pkey; Type: CONSTRAINT; Schema: public; Owner: youtube
--

ALTER TABLE ONLY public.video
    ADD CONSTRAINT video_pkey PRIMARY KEY (id);


--
-- TOC entry 2048 (class 1259 OID 121676)
-- Name: video_idpl_idx; Type: INDEX; Schema: public; Owner: youtube
--

CREATE INDEX video_idch_idx ON public.video USING btree (idpl);


--
-- TOC entry 2051 (class 2606 OID 121677)
-- Name: metrics_idv_fkey; Type: FK CONSTRAINT; Schema: public; Owner: youtube
--

ALTER TABLE ONLY public.metric
    ADD CONSTRAINT metrics_idv_fkey FOREIGN KEY (idvideo) REFERENCES public.video(id);


--
-- TOC entry 2052 (class 2606 OID 121682)
-- Name: video_idpl_fkey; Type: FK CONSTRAINT; Schema: public; Owner: youtube
--

ALTER TABLE ONLY public.video
    ADD CONSTRAINT video_idch_fkey FOREIGN KEY (idpl) REFERENCES public.channel(id);


--
-- TOC entry 2174 (class 0 OID 0)
-- Dependencies: 7
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: youtube
--

REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM youtube;
GRANT ALL ON SCHEMA public TO youtube;
GRANT ALL ON SCHEMA public TO PUBLIC;


--
-- TOC entry 2176 (class 0 OID 0)
-- Dependencies: 198
-- Name: FUNCTION return_metrics(_idv character); Type: ACL; Schema: public; Owner: youtube
--

REVOKE ALL ON FUNCTION public.return_metrics(_idv character) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.return_metrics(_idv character) FROM youtube;
GRANT ALL ON FUNCTION public.return_metrics(_idv character) TO youtube;
GRANT ALL ON FUNCTION public.return_metrics(_idv character) TO postgres;
GRANT ALL ON FUNCTION public.return_metrics(_idv character) TO PUBLIC;


--
-- TOC entry 2177 (class 0 OID 0)
-- Dependencies: 199
-- Name: FUNCTION return_metrics(_idv character, _from_ch character, _to_ch character); Type: ACL; Schema: public; Owner: youtube
--

REVOKE ALL ON FUNCTION public.return_metrics(_idv character, _from_ch character, _to_ch character) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.return_metrics(_idv character, _from_ch character, _to_ch character) FROM youtube;
GRANT ALL ON FUNCTION public.return_metrics(_idv character, _from_ch character, _to_ch character) TO youtube;
GRANT ALL ON FUNCTION public.return_metrics(_idv character, _from_ch character, _to_ch character) TO postgres;
GRANT ALL ON FUNCTION public.return_metrics(_idv character, _from_ch character, _to_ch character) TO PUBLIC;


--
-- TOC entry 2178 (class 0 OID 0)
-- Dependencies: 197
-- Name: FUNCTION return_video(_idv character, OUT _idpl character, OUT _title character, OUT _description character, OUT _chtitle character, OUT _chid character, OUT _publishedat timestamp with time zone, OUT _count_metrics integer, OUT _min_timemetric timestamp with time zone, OUT _max_timemetric timestamp with time zone); Type: ACL; Schema: public; Owner: youtube
--

REVOKE ALL ON FUNCTION public.return_video(_idv character, OUT _idpl character, OUT _title character, OUT _description character, OUT _chtitle character, OUT _chid character, OUT _publishedat timestamp with time zone, OUT _count_metrics integer, OUT _min_timemetric timestamp with time zone, OUT _max_timemetric timestamp with time zone) FROM PUBLIC;
REVOKE ALL ON FUNCTION public.return_video(_idv character, OUT _idpl character, OUT _title character, OUT _description character, OUT _chtitle character, OUT _chid character, OUT _publishedat timestamp with time zone, OUT _count_metrics integer, OUT _min_timemetric timestamp with time zone, OUT _max_timemetric timestamp with time zone) FROM youtube;
GRANT ALL ON FUNCTION public.return_video(_idv character, OUT _idpl character, OUT _title character, OUT _description character, OUT _chtitle character, OUT _chid character, OUT _publishedat timestamp with time zone, OUT _count_metrics integer, OUT _min_timemetric timestamp with time zone, OUT _max_timemetric timestamp with time zone) TO youtube;
GRANT ALL ON FUNCTION public.return_video(_idv character, OUT _idpl character, OUT _title character, OUT _description character, OUT _chtitle character, OUT _chid character, OUT _publishedat timestamp with time zone, OUT _count_metrics integer, OUT _min_timemetric timestamp with time zone, OUT _max_timemetric timestamp with time zone) TO postgres;
GRANT ALL ON FUNCTION public.return_video(_idv character, OUT _idpl character, OUT _title character, OUT _description character, OUT _chtitle character, OUT _chid character, OUT _publishedat timestamp with time zone, OUT _count_metrics integer, OUT _min_timemetric timestamp with time zone, OUT _max_timemetric timestamp with time zone) TO PUBLIC;


--
-- TOC entry 2179 (class 0 OID 0)
-- Dependencies: 181
-- Name: TABLE metric; Type: ACL; Schema: public; Owner: youtube
--

REVOKE ALL ON TABLE public.metric FROM PUBLIC;
REVOKE ALL ON TABLE public.metric FROM youtube;
GRANT ALL ON TABLE public.metric TO youtube;
GRANT ALL ON TABLE public.metric TO postgres;


--
-- TOC entry 2181 (class 0 OID 0)
-- Dependencies: 182
-- Name: SEQUENCE metrics_id_seq; Type: ACL; Schema: public; Owner: youtube
--

REVOKE ALL ON SEQUENCE public.metrics_id_seq FROM PUBLIC;
REVOKE ALL ON SEQUENCE public.metrics_id_seq FROM youtube;
GRANT ALL ON SEQUENCE public.metrics_id_seq TO youtube;
GRANT ALL ON SEQUENCE public.metrics_id_seq TO postgres;


--
-- TOC entry 2182 (class 0 OID 0)
-- Dependencies: 183
-- Name: TABLE playlist; Type: ACL; Schema: public; Owner: youtube
--

REVOKE ALL ON TABLE public.playlist FROM PUBLIC;
REVOKE ALL ON TABLE public.playlist FROM youtube;
GRANT ALL ON TABLE public.playlist TO youtube;
GRANT ALL ON TABLE public.playlist TO postgres;


--
-- TOC entry 2183 (class 0 OID 0)
-- Dependencies: 184
-- Name: TABLE video; Type: ACL; Schema: public; Owner: youtube
--

REVOKE ALL ON TABLE public.video FROM PUBLIC;
REVOKE ALL ON TABLE public.video FROM youtube;
GRANT ALL ON TABLE public.video TO youtube;
GRANT ALL ON TABLE public.video TO postgres;


-- Completed on 2018-11-14 16:45:09 EET

--
-- PostgreSQL database dump complete
--

