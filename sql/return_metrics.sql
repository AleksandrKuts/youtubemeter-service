﻿/* Повертає дані метрик по заданому відео. Рядки повертаються в заданої кількості (_MAX_RETURN_COUNT_ROWS CONSTANT) рівномірно 
   розподілені по інтервалу запиту, плюс перший та останній запис. Розподіляємо орієнтуючись на номери записів в Select, 
   для цього використовуємо ROW_NUMBER()

   Наприклад, при обмеженні на повернення 3 записів повернеться: 

         повна вибірка           буде повернуто
      
   {0,1,2,3,4,5,6,7,8,9}       =>  {0,3,6,9}
   {0,1,2,3,4,5,6,7,8,9,10}    =>  {0,3,6,10}
   {0,1,2,3,4,5,6,7,8,9,10,11} =>  {0,3,6,11}
   {0,1,2,3,4,5,6,7,8,9,10,12} =>  {0,4,8,12}

*/
CREATE OR REPLACE FUNCTION public.return_metrics(
    IN _idv character) /* id відео */
  RETURNS TABLE(commentcount bigint, likecount bigint, dislikecount bigint, viewcount bigint, timemetric timestamp with time zone) AS
$BODY$
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
$BODY$
  LANGUAGE plpgsql 