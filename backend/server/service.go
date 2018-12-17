package server

import (
	"encoding/json"
	"strconv"
	"errors"
	"github.com/AleksandrKuts/youtubemeter-service/backend/config"
	"github.com/hashicorp/golang-lru"
	"time"
)

var MIN_TIME = time.Time{}

// Кеш для відео опису
var cacheVideoDescription *lru.TwoQueueCache

// Кеш для відео 
var cacheVideos *lru.TwoQueueCache

// Кеш для плейлистів
var cachePlayLists *lru.TwoQueueCache

// Кеш для списку плейлистів
var listCachePlayLists *ListPlayListInCache

// Глобальні метрики
var globalCounts *GlobalCounts

func init() {
	var err error

	if *config.EnableCache {
		cacheVideos, err = lru.New2Q(*config.MaxSizeCacheVideo)
		if err != nil {
			log.Fatalf("err: %v", err)
		}

		cacheVideoDescription, err = lru.New2Q(*config.MaxSizeCacheVideoDescription)
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
	log.Debugf("getVideoById(id: %v)", id)

	if id == "" {
		return nil, errors.New("video id is null")
	}

	var ok bool = false
	var videoi interface{}

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		videoi, ok = cacheVideoDescription.Get(id)
		log.Debugf("id: %v, cache, have data? %v", id, ok)

		// Перевіряємо чи є дані в кеші
		if ok {
			video := videoi.(*VideoInCache)
			log.Debugf("id: %v, metrics: %v, video: %v, published: %v", id, video.updateMetrics, video.updateVideo, video.publishedAt)

			// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
			// ніж період збору метрик, або якщо збір метрик вже припинився.
			if video.videoResponce != nil && 
				( time.Since(video.publishedAt) > *config.PeriodCollectionCache ||
				time.Since(video.updateVideo) < *config.PeriodMeterCache) {

				log.Infof("id: %v, get video from cache", id)
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
			cacheVideoDescription.Add(id, &VideoInCache{MIN_TIME, time.Now(), youtubeVideo.PublishedAt, stringJsonVideo, nil})
			log.Debugf("id: %v, cache, add video, published: %v", id, youtubeVideo.PublishedAt)
		}
	}

	log.Infof("id: %v, get video skip cache", id)
	return stringJsonVideo, nil
}

// Отримати метрики по відео id за заданий період
// Такий запит не використовуэ кеш
func getMetricsByIdFromTo(id string, from, to string) ([]byte, error) {
	log.Debugf("getMetricsByIdFromTo(id: %v, from %v, to %v) ", id, from, to)

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
	log.Debugf("getMetricsById(id: %v, from: %v, to: %v)", id, from, to)
	if id == "" {
		return nil, errors.New("video id is null")
	}

	// Период заданий, такий запит обробляємо окремо
	if from != "" || to != "" {
		return getMetricsByIdFromTo(id, from, to)
	}

	var ok bool = false
	var videoi interface{}

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		videoi, ok = cacheVideoDescription.Get(id)
		log.Debugf("id: %v, cache, have data? %v", id, ok)

		// Перевіряємо чи є дані в кеші
		if ok {
			video := videoi.(*VideoInCache)
			log.Debugf("id: %v, metrics: %v, video: %v, published: %v", id, video.updateMetrics, video.updateVideo, video.publishedAt)

			// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
			// ніж період збору метрик, або якщо збір метрик вже припинився.
			if video.metricsResponce != nil && 
				( time.Since(video.updateMetrics) < *config.PeriodMeterCache ||
				time.Since(video.publishedAt) > *config.PeriodCollectionCache) {

				log.Infof("id: %v, get metrics from cache", id)
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
			cacheVideoDescription.Add(id, &VideoInCache{time.Now(), MIN_TIME, time.Now(), nil, metricsVideoJson})
			log.Debugf("id: %v, cache, add metrics", id)
		}
	}

	log.Infof("id: %v, get metrics skip cache", id)
	return metricsVideoJson, nil
}

// Отримати список відео
func getVideos(offset int) ([]byte, error) {
	log.Debugf("getVideos(offset: %v)", offset)
	cacheId := offset

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		videosi, ok := cacheVideos.Get(cacheId)
		log.Debugf("offset: %v, cache, have data? %v", cacheId, ok)

		// Перевіряємо чи є дані в кеші
		if ok {
			videos := videosi.(*YoutubeVideoShortInCache)
			log.Debugf("offset: %v, timeUpdate: %v", cacheId, videos.timeUpdate)

			// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
			// ніж період перевірки списку відео в плейлисті
			if time.Since(videos.timeUpdate) < *config.PeriodVideoCache {
				log.Debugf("offset: %v, cache, videos: %v", cacheId, string(videos.responce))

				log.Infof("offset: %v, get videos from cache", cacheId)
				return videos.responce, nil
			}
			log.Debugf("offset: %v, cache, skip", cacheId)
		}

	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	stringVideos, err := getVideosByPlayListIdFromDB("", offset)
	if err != nil {
		return nil, err
	}

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		// Додаємо запит до кешу
		cacheVideos.Add(cacheId, &YoutubeVideoShortInCache{time.Now(), stringVideos})
		log.Debugf("offset: %v, cache, add videos", cacheId)
	}

	log.Infof("offset: %v, get videos skip cache", cacheId)
	return stringVideos, nil
}


// Отримати список відео по id плейлиста
func getVideosByPlayListId(id string, offset int) ([]byte, error) {
	log.Debugf("getVideosByPlayListId(id: %v, offset: %v)", id, offset)

	if id == "" {
		return nil, errors.New("video id is null")
	}

	cacheId := id + "_" + strconv.Itoa(offset)

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		playlisti, ok := cachePlayLists.Get(cacheId)
		log.Debugf("id: %v, cache, have data? %v", cacheId, ok)

		// Перевіряємо чи є дані в кеші
		if ok {
			playlist := playlisti.(*YoutubeVideoShortInCache)
			log.Debugf("id: %v, timeUpdate: %v", cacheId, playlist.timeUpdate)

			// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
			// ніж період перевірки списку відео в плейлисті
			if time.Since(playlist.timeUpdate) < *config.PeriodVideoCache {
				log.Infof("id: %v, get videos for playlist from cache", cacheId)
				log.Debugf("id: %v, cache, playlist: %v", cacheId, string(playlist.responce))

				return playlist.responce, nil
			}
			log.Debugf("id: %v, cache, skip", cacheId)
		}

	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	stringVideos, err := getVideosByPlayListIdFromDB(id, offset)
	if err != nil {
		return nil, err
	}

	// з кешем робимо тільки якщо він включений
	if *config.EnableCache {
		// Додаємо запит до кешу
		cachePlayLists.Add(cacheId, &YoutubeVideoShortInCache{time.Now(), stringVideos})
		log.Debugf("id: %v, cache, add playlists", cacheId)
	}

	log.Infof("id: %v, get videos for playlist skip cache", cacheId)
	return stringVideos, nil
}

// Отримати список плейлистів
func getPlaylists(onlyEnable bool) ([]byte, error) {
	log.Debugf("getPlaylists(onlyEnable: %v)", onlyEnable)
	// з кешем робимо тільки якщо він включений та це не запит адміністратора на всі плейлисти
	if *config.EnableCache && onlyEnable {
		log.Debugf("playlists, cache, timeUpdate: %v", listCachePlayLists.timeUpdate)
		
		// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
		// ніж період перевірки списку плейлистів
		if time.Since(listCachePlayLists.timeUpdate) < *config.PeriodPlayListCache {
			log.Debugf("playlists, cache, list playlists: %v", string(listCachePlayLists.responce))
			log.Info("get playlist from cache")

			return listCachePlayLists.responce, nil
		}
		log.Debug("playlists, cache, skip")
	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	response, err := getPlaylistsFromDB(onlyEnable)
	if err != nil {
		return nil, err
	}

	responcePlayList := ResponcePlayList{*config.MaxViewVideosInPlayLists, response}
	
	// Конвертуємо відповідь в json-формат
	stringJsonPlaylists, err := json.Marshal(&responcePlayList)

	if err != nil {
		log.Errorf("Error convert select to Playlists: response=%v, error=%v", stringJsonPlaylists, err)
		return nil, err
	}

	log.Debugf("playlists: %v", string(stringJsonPlaylists))

	// з кешем робимо тільки якщо він включений та це не запит адміністратора на всі плейлисти
	if *config.EnableCache && onlyEnable {
		// Додаємо запит до кешу
		listCachePlayLists.update(stringJsonPlaylists)
		log.Debugf("playlists, cache, add list playlists")
	}

	log.Info("get playlist skip cache")
	return stringJsonPlaylists, nil
}

func getGlobalCounts(version string) ([]byte, error) {
	log.Debug("get globalCounts")
	if globalCounts == nil || time.Since(globalCounts.TimeUpdate) > *config.PeriodVideoCache {
		g, err := getGlobalCountsFromDB(version)
		if err == nil {
			globalCounts = g;
		}		
	} 		

	// Конвертуємо відповідь в json-формат
	stringGlobalCounts, err := json.Marshal(&globalCounts)

	if err != nil {
		log.Errorf("Error convert select to Playlists: response=%v, error=%v", globalCounts, err)
		return nil, err
	}
	
	log.Debugf("globalCounts: %v", string(stringGlobalCounts))
	
	return stringGlobalCounts, nil
}
