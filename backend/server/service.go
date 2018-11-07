package server

import (
	"encoding/json"
	"errors"
	"github.com/AleksandrKuts/youtubemeter-service/backend/config"
	"github.com/hashicorp/golang-lru"
	"time"
)

// Кеш для відео
var cacheVideos *lru.TwoQueueCache

// Кеш для плейлистів
var cachePlayLists *lru.TwoQueueCache

// Кеш для списку плейлистів
var listCachePlayLists *ListPlayListInCache

var MIN_TIME = time.Time{}

func init() {
	var err error

	if *config.EnableCache {
		cacheVideos, err = lru.New2Q(*config.MaxSizeCacheVideo)
		if err != nil {
			log.Fatalf("err: %v", err)
		}

		cachePlayLists, err = lru.New2Q(*config.MaxSizeCachePlaylists)
		if err != nil {
			log.Fatalf("err: %v", err)
		}

		listCachePlayLists = &ListPlayListInCache{MIN_TIME, nil}
	}
}

// Додати плей-лист
func addPlayList(playlist *PlayList) error {
	// Список плейлистів в кеші треба буде оновити
	listCachePlayLists.reset()

	return addPlayListDB(playlist)
}

// Оновити плей-лист
func updatePlayList(id string, playlist *PlayList) error {
	// Список плейлистів в кеші треба буде оновити
	listCachePlayLists.reset()

	return updatePlayListDB(id, playlist)
}

// Видалити плей-лист
func deletePlayList(playlistId string) error {
	// Список плейлистів в кеші треба буде оновити
	listCachePlayLists.reset()

	return deletePlayListDB(playlistId)
}

// Отримати опис відео по його id
func getVideoById(id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("video id is null")
	}
	log.Debugf("id: %v", id)

	var ok bool = false
	var videoi interface{}

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		videoi, ok = cacheVideos.Get(id)
		log.Debugf("id: %v, cache, have data? %v", id, ok)

		// Перевіряємо чи є дані в кеші
		if ok {
			video := videoi.(*VideoInCache)
			log.Debugf("id: %v, metrics: %v, video: %v, published: %v", id, video.updateMetrics, video.updateVideo, video.publishedAt)

			// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
			// ніж період збору метрик, або якщо збір метрик вже припинився.
			if time.Since(video.publishedAt) > *config.PeriodCollectionCache ||
				time.Since(video.updateVideo) < *config.PeriodMeterCache {

				log.Debugf("id: %v, cache, video %v", id, string(video.videoResponce))

				return video.videoResponce, nil
			}
			log.Debugf("id: %v, cache, skip", id)
		}
	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	youtubeVideo, err := getVideoByIdFromDB(id)
	if err != nil {
		return nil, err
	}

	// Конвертуємо відповідь в json-формат
	stringJsonVideo, err := json.Marshal(*youtubeVideo)
	if err != nil {
		log.Errorf("Error convert select to video: response=%v, error=%v", *youtubeVideo, err)
		return nil, err
	}
	log.Debugf("id: %v, video=%v", id, string(stringJsonVideo))

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		// Якщо дані по запиту вже в кеші, то тільки корегуємо їх
		if ok {
			// Корегуємо дані в кеші
			videoi.(*VideoInCache).updateCacheVideo(youtubeVideo.PublishedAt, stringJsonVideo)
			log.Debugf("id: %v, cache, update video, published: %v", id, youtubeVideo.PublishedAt)
		} else {
			// Додаємо запит до кешу
			cacheVideos.Add(id, &VideoInCache{MIN_TIME, time.Now(), youtubeVideo.PublishedAt, stringJsonVideo, nil})
			log.Debugf("id: %v, cache, add video, published: %v", id, youtubeVideo.PublishedAt)
		}
	}

	return stringJsonVideo, nil
}

// Отримати метрики по відео id за заданий період
// Такий запит не використовуэ кеш
func getMetricsByIdFromTo(id string, from, to string) ([]byte, error) {
	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	response, err := getMetricsByIdFromDB(id, from, to)
	if err != nil {
		return nil, err
	}

	// Конвертуємо відповідь в json-формат
	metricsVideoJson, err := json.Marshal(response)
	if err != nil {
		log.Errorf("Error convert select to Metrics: response=%v, error=%v", response, err)
		return nil, err
	}
	log.Debugf("id: %v, metrics=%v", id, string(metricsVideoJson))

	return metricsVideoJson, nil
}

// Отримати метрики по відео id або за весь період, або за заданий період
// Якщо період не заданий то використовується кеш, якщо період заданий кеш не використовується
func getMetricsById(id string, from, to string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("video id is null")
	}
	log.Debugf("id: %v, from: %v, to: %v", id, from, to)

	// Период заданий, такий запит обробляємо окремо
	if from != "" || to != "" {
		return getMetricsByIdFromTo(id, from, to)
	}

	var ok bool = false
	var videoi interface{}

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		videoi, ok = cacheVideos.Get(id)
		log.Debugf("id: %v, cache, have data? %v", id, ok)

		// Перевіряємо чи є дані в кеші
		if ok {
			video := videoi.(*VideoInCache)
			log.Debugf("id: %v, metrics: %v, video: %v, published: %v", id, video.updateMetrics, video.updateVideo, video.publishedAt)

			// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
			// ніж період збору метрик, або якщо збір метрик вже припинився.
			if time.Since(video.updateMetrics) < *config.PeriodMeterCache ||
				time.Since(video.publishedAt) > *config.PeriodCollectionCache {

				log.Debugf("id: %v, cache, metrics: %v", id, string(video.metricsResponce))

				return video.metricsResponce, nil
			}
			log.Debugf("id: %v, cache, skip", id)
		}

	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	response, err := getMetricsByIdFromDB(id, from, to)
	if err != nil {
		return nil, err
	}

	// Конвертуємо відповідь в json-формат
	metricsVideoJson, err := json.Marshal(response)
	if err != nil {
		log.Errorf("Error convert select to Metrics: response=%v, error=%v", response, err)
		return nil, err
	}
	log.Debugf("id: %v, metrics=%v", id, string(metricsVideoJson))

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		// Якщо дані по запиту вже в кеші, то тільки корегуємо їх
		if ok {
			// Корегуємо дані в кеші
			videoi.(*VideoInCache).updateCacheMetrics(metricsVideoJson)
			log.Debugf("id: %v, cache, update metrics, published", id)
		} else {
			// Додаємо запит до кешу
			cacheVideos.Add(id, &VideoInCache{time.Now(), MIN_TIME, time.Now(), nil, metricsVideoJson})
			log.Debugf("id: %v, cache, add metrics", id)
		}
	}

	return metricsVideoJson, nil
}

// Отримати список відео по id плейлиста
func getVideosByPlayListId(id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("video id is null")
	}

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		playlisti, ok := cachePlayLists.Get(id)
		log.Debugf("id: %v, cache, have data? %v", id, ok)

		// Перевіряємо чи є дані в кеші
		if ok {
			playlist := playlisti.(*PlayListInCache)
			log.Debugf("id: %v, timeUpdate: %v", id, playlist.timeUpdate)

			// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
			// ніж період перевірки списку відео в плейлисті
			if time.Since(playlist.timeUpdate) < *config.PeriodVideoCache {
				log.Debugf("id: %v, cache, playlist: %v", id, string(playlist.responce))

				return playlist.responce, nil
			}
			log.Debugf("id: %v, cache, skip", id)
		}

	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	stringVideos, err := getVideosByPlayListIdFromDB(id)
	if err != nil {
		return nil, err
	}

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		// Додаємо запит до кешу
		cachePlayLists.Add(id, &PlayListInCache{time.Now(), stringVideos})
		log.Debugf("id: %v, cache, add playlists", id)
	}

	return stringVideos, nil
}

// Отримати список плейлистів
func getPlaylists(onlyEnable bool) ([]byte, error) {
	// з кешем робимо тільки якщо він включений та це не запит адміністратора на всі плейлисти
	if *config.EnableCache && onlyEnable {
		// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
		// ніж період перевірки списку плейлистів
		if time.Since(listCachePlayLists.timeUpdate) < *config.PeriodPlayListCache {
			log.Debugf("cache, list playlists: %v", string(listCachePlayLists.responce))

			return listCachePlayLists.responce, nil
		}
	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	stringPlaylists, err := getPlaylistsFromDB(onlyEnable)
	if err != nil {
		return nil, err
	}

	// з кешем робимо тільки якщо він включений та це не запит адміністратора на всі плейлисти
	if *config.EnableCache && onlyEnable {
		// Додаємо запит до кешу
		listCachePlayLists.update(stringPlaylists)
		log.Debugf("cache, add list playlists")
	}

	return stringPlaylists, nil
}
