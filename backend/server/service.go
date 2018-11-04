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

	youtubeVideo, err := getVideoByIdFromDB(id)
	if err != nil {
		return nil, err
	}
	
	stringJsonVideo, err := json.Marshal(*youtubeVideo)

	if err != nil {
		log.Errorf("Error convert select to Metrics: response=%v, error=%v", *youtubeVideo, err)
		return nil, err
	}

	log.Debugf("Video=%v", string(stringJsonVideo))

	return stringJsonVideo, nil
}

// Отримати метрики по відео id за заданий період
func getMetricsById(id string, from, to string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("video id is null")
	}

	// Перевірка чи є дані в кеші. В кеші зберігаються тільки запроси за весь період
	if from == "" && to == "" {
		metricsi, ok := cacheMetrics.Get(id)

		// Якщо дані в кеші є, то беремо їх тільки якщо з останнього запиту пройшло часу менш
		// ніж період збору метрик, або якщо збір метрик вже припинився.
		if ok {
			metrics := metricsi.(MetricsInCache)
			if time.Since(metrics.create) < *config.PeriodMeterCache {
				log.Debug("get video from cache")

				return metrics.responce, nil
			}
		}
	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	response, err := getMetricsByIdFromDB(id, from, to)
	if err != nil {
		return nil, err
	}

	metricsVideoJson, err := json.Marshal(*response)

	// Зберігаємо запит в кеші. В кеші зберігаються тільки запроси за весь період
	if from == "" && to == "" {
		cacheMetrics.Add(id, MetricsInCache{time.Now(), metricsVideoJson})
		log.Debug("set video to cache")
	}

 	if err != nil {
		log.Errorf("Error convert select to Metrics: response=%v, error=%v", response, err)
		return nil, err
	}

	log.Debugf("Metrics=%v", string(metricsVideoJson))

	return metricsVideoJson, nil
}
