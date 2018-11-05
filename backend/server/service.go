package server

import (
	"errors"
	"encoding/json"
	"github.com/hashicorp/golang-lru"
	"time"
	"github.com/AleksandrKuts/youtubemeter-service/backend/config"	
)

// Кеш для метрик
var cacheMetrics *lru.TwoQueueCache

// Кеш для відео
var cacheVideo *lru.TwoQueueCache

func init() {
	var err error

	cacheMetrics, err = lru.New2Q(*config.MaxSizeCacheMetrics)
	if err != nil {
		log.Fatalf("err: %v", err)
	}

	cacheVideo, err = lru.New2Q(*config.MaxSizeCacheVideo)
	if err != nil {
		log.Fatalf("err: %v", err)
	}	
}

// Отримати опис відео по його id
func getVideoById(id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("video id is null")
	}
	log.Debugf("id: %v", id)

	videoi, ok := cacheVideo.Get(id)
	log.Debugf("id: %v, cache, try get video: %v", id, ok)

	// Якщо дані в кеші є, то беремо їх тільки якщо з останнього запиту пройшло часу менш
	// ніж період збору метрик, або якщо збір метрик вже припинився.
	if ok {
		
		video := videoi.(*VideoInCache)
		if time.Since(video.create) < *config.PeriodMeterCache ||
		   time.Since(video.publishedAt) > *config.PeriodCollectionCache {
		   	
			log.Debugf("id: %v, cache, video %v", id, string(video.responce) )

			return video.responce, nil
		}
	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	youtubeVideo, err := getVideoByIdFromDB(id)
	if err != nil {
		return nil, err
	}
	
	stringJsonVideo, err := json.Marshal(*youtubeVideo)
	
	// Додаємо запит до кешу
	cacheVideo.Add(id, &VideoInCache{time.Now(), youtubeVideo.PublishedAt, stringJsonVideo})
	log.Debugf("id: %v, cache, add video, published: %v", id, youtubeVideo.PublishedAt)

	if err != nil {
		log.Errorf("Error convert select to video: response=%v, error=%v", *youtubeVideo, err)
		return nil, err
	}

	log.Debugf("id: %v, video=%v", id, string(stringJsonVideo))

	return stringJsonVideo, nil
}

// Отримати метрики по відео id за заданий період
func getMetricsById(id string, from, to string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("video id is null")
	}
	log.Debugf("id: %v, from: %v, to: %v", id, from, to)

	// Перевірка чи є дані в кеші. В кеші зберігаються тільки запроси за весь період
	if from == "" && to == "" {
		metricsi, ok := cacheMetrics.Get(id)
		log.Debugf("id: %v, cache, have data?: %v", id, ok)

		// Якщо дані в кеші є, то беремо їх тільки якщо з останнього запиту пройшло часу менш
		// ніж період збору метрик, або якщо збір метрик вже припинився.
		if ok {
			metrics := metricsi.(*MetricsInCache)
			if time.Since(metrics.create) < *config.PeriodMeterCache ||
			   time.Since(metrics.publishedAt) > *config.PeriodCollectionCache {
		
				log.Debugf("id: %v, cache, metrics: %v", id, string(metrics.responce))

				return metrics.responce, nil
			}
			log.Debug("cache, skip")
		}
	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	response, err := getMetricsByIdFromDB(id, from, to)
	if err != nil {
		return nil, err
	}

	metricsVideoJson, err := json.Marshal(response)

	// Зберігаємо запит в кеші. В кеші зберігаються тільки запроси за весь період
	if from == "" && to == "" {
		
		// Відомості про публікацію беремо з кеша відео
		published := time.Now();		
		videoi, ok := cacheVideo.Peek(id)
		if ok {
			published = videoi.(*VideoInCache).publishedAt
			log.Debugf("id: %v, cache, get video pablished: %v", id, published)
		}
		
		cacheMetrics.Add(id, &MetricsInCache{time.Now(), published, metricsVideoJson})
		log.Debugf("id: %v, cache, add metrics: published: %v", id, published)
	}

 	if err != nil {
		log.Errorf("Error convert select to Metrics: response=%v, error=%v", response, err)
		return nil, err
	}

	log.Debugf("id: %v, metrics=%v", id, string(metricsVideoJson))

	return metricsVideoJson, nil
}
