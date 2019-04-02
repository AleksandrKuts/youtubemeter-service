package backend

import (
	"encoding/json"
	"strconv"
	"errors"
	"github.com/hashicorp/golang-lru"
	"time"
)

var MIN_TIME = time.Time{}

// Кеш для відео опису
var cacheVideoDescription *lru.TwoQueueCache

// Кеш для відео 
var cacheVideos *lru.TwoQueueCache

// Кеш для каналів
var cacheChannels *lru.TwoQueueCache

// Кеш для списку каналів
var listCacheChannels *ListChannelInCache

// Глобальні метрики
var globalCounts *GlobalCounts

func init() {
	var err error

	if *EnableCache {
		cacheVideos, err = lru.New2Q(*MaxSizeCacheVideo)
		if err != nil {
			Logger.Fatalf("err: %v", err)
		}

		cacheVideoDescription, err = lru.New2Q(*MaxSizeCacheVideoDescription)
		if err != nil {
			Logger.Fatalf("err: %v", err)
		}

		cacheChannels, err = lru.New2Q(*MaxSizeCacheChannels)
		if err != nil {
			Logger.Fatalf("err: %v", err)
		}

		listCacheChannels = &ListChannelInCache{MIN_TIME, nil}
	}
}

// Додати канал
func addChannel(channel *Channel) error {
	// Список каналів в кеші треба буде оновити
	listCacheChannels.reset()

	return addChannelDB(channel)
}

// Оновити канал
func updateChannel(id string, channel *Channel) error {
	// Список каналів в кеші треба буде оновити
	listCacheChannels.reset()

	return updateChannelDB(id, channel)
}

// Видалити канал
func deleteChannel(channelId string) error {
	// Список каналів в кеші треба буде оновити
	listCacheChannels.reset()

	return deleteChannelDB(channelId)
}

// Отримати опис відео по його id
func getVideoById(id string) ([]byte, error) {
	Logger.Debugf("getVideoById(id: %v)", id)

	if id == "" {
		return nil, errors.New("video id is null")
	}

	var ok bool = false
	var videoi interface{}

	// з кешем робимо тільки якщо він включений
	if *EnableCache {
		videoi, ok = cacheVideoDescription.Get(id)
		Logger.Debugf("id: %v, cache, have data? %v", id, ok)

		// Перевіряємо чи є дані в кеші
		if ok {
			video := videoi.(*VideoInCache)
			Logger.Debugf("id: %v, metrics: %v, video: %v, published: %v", id, video.updateMetrics, video.updateVideo, video.publishedAt)

			// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
			// ніж період збору метрик, або якщо збір метрик вже припинився.
			if video.videoResponce != nil && 
				( time.Since(video.publishedAt) > *PeriodCollectionCache ||
				time.Since(video.updateVideo) < *PeriodMeterCache) {

				Logger.Infof("id: %v, get video from cache", id)
				Logger.Debugf("id: %v, cache, video %v", id, string(video.videoResponce))

				return video.videoResponce, nil
			}
			Logger.Debugf("id: %v, cache, skip", id)
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
		Logger.Errorf("Error convert select to video: response=%v, error=%v", *youtubeVideo, err)
		return nil, err
	}
	Logger.Debugf("id: %v, video=%v", id, string(stringJsonVideo))

	// з кешем робимо тільки якщо він включений
	if *EnableCache {
		// Якщо дані по запиту вже в кеші, то тільки корегуємо їх
		if ok {
			// Корегуємо дані в кеші
			videoi.(*VideoInCache).updateCacheVideo(youtubeVideo.PublishedAt, stringJsonVideo)
			Logger.Debugf("id: %v, cache, update video, published: %v", id, youtubeVideo.PublishedAt)
		} else {
			// Додаємо запит до кешу
			cacheVideoDescription.Add(id, &VideoInCache{MIN_TIME, time.Now(), youtubeVideo.PublishedAt, stringJsonVideo, nil})
			Logger.Debugf("id: %v, cache, add video, published: %v", id, youtubeVideo.PublishedAt)
		}
	}

	Logger.Infof("id: %v, get video skip cache", id)
	return stringJsonVideo, nil
}

// Отримати метрики по відео id за заданий період
// Такий запит не використовуэ кеш
func getMetricsByIdFromTo(id string, from, to string) ([]byte, error) {
	Logger.Debugf("getMetricsByIdFromTo(id: %v, from %v, to %v) ", id, from, to)

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	response, err := getMetricsByIdFromDB(id, from, to)
	if err != nil {
		return nil, err
	}

	// Конвертуємо відповідь в json-формат
	metricsVideoJson, err := json.Marshal(response)
	if err != nil {
		Logger.Errorf("Error convert select to Metrics: response=%v, error=%v", response, err)
		return nil, err
	}
	Logger.Debugf("id: %v, metrics=%v", id, string(metricsVideoJson))

	return metricsVideoJson, nil
}

// Отримати метрики по відео id або за весь період, або за заданий період
// Якщо період не заданий то використовується кеш, якщо період заданий кеш не використовується
func getMetricsById(id string, from, to string) ([]byte, error) {
	Logger.Debugf("getMetricsById(id: %v, from: %v, to: %v)", id, from, to)
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
	if *EnableCache {
		videoi, ok = cacheVideoDescription.Get(id)
		Logger.Debugf("id: %v, cache, have data? %v", id, ok)

		// Перевіряємо чи є дані в кеші
		if ok {
			video := videoi.(*VideoInCache)
			Logger.Debugf("id: %v, metrics: %v, video: %v, published: %v", id, video.updateMetrics, video.updateVideo, video.publishedAt)

			// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
			// ніж період збору метрик, або якщо збір метрик вже припинився.
			if video.metricsResponce != nil && 
				( time.Since(video.updateMetrics) < *PeriodMeterCache ||
				time.Since(video.publishedAt) > *PeriodCollectionCache) {

				Logger.Infof("id: %v, get metrics from cache", id)
				Logger.Debugf("id: %v, cache, metrics: %v", id, string(video.metricsResponce))

				return video.metricsResponce, nil
			}
			Logger.Debugf("id: %v, cache, skip", id)
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
		Logger.Errorf("Error convert select to Metrics: response=%v, error=%v", response, err)
		return nil, err
	}
	Logger.Debugf("id: %v, metrics=%v", id, string(metricsVideoJson))

	// з кешем робимо тільки якщо він включений
	if *EnableCache {
		// Якщо дані по запиту вже в кеші, то тільки корегуємо їх
		if ok {
			// Корегуємо дані в кеші
			videoi.(*VideoInCache).updateCacheMetrics(metricsVideoJson)
			Logger.Debugf("id: %v, cache, update metrics, published", id)
		} else {
			// Додаємо запит до кешу
			cacheVideoDescription.Add(id, &VideoInCache{time.Now(), MIN_TIME, time.Now(), nil, metricsVideoJson})
			Logger.Debugf("id: %v, cache, add metrics", id)
		}
	}

	Logger.Infof("id: %v, get metrics skip cache", id)
	return metricsVideoJson, nil
}

// Отримати список відео
func getVideos(offset int) ([]byte, error) {
	Logger.Debugf("getVideos(offset: %v)", offset)
	cacheId := offset

	// з кешем робимо тільки якщо він включений
	if *EnableCache {
		videosi, ok := cacheVideos.Get(cacheId)
		Logger.Debugf("offset: %v, cache, have data? %v", cacheId, ok)

		// Перевіряємо чи є дані в кеші
		if ok {
			videos := videosi.(*YoutubeVideoShortInCache)
			Logger.Debugf("offset: %v, timeUpdate: %v", cacheId, videos.timeUpdate)

			// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
			// ніж період перевірки списку відео  
			if time.Since(videos.timeUpdate) < *PeriodVideoCache {
				Logger.Debugf("offset: %v, cache, videos: %v", cacheId, string(videos.responce))

				Logger.Infof("offset: %v, get videos from cache", cacheId)
				return videos.responce, nil
			}
			Logger.Debugf("offset: %v, cache, skip", cacheId)
		}

	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	stringVideos, err := getVideosByChannelIdFromDB("", offset)
	if err != nil {
		return nil, err
	}

	// з кешем робимо тільки якщо він включений
	if *EnableCache {
		// Додаємо запит до кешу
		cacheVideos.Add(cacheId, &YoutubeVideoShortInCache{time.Now(), stringVideos})
		Logger.Debugf("offset: %v, cache, add videos", cacheId)
	}

	Logger.Infof("offset: %v, get videos skip cache", cacheId)
	return stringVideos, nil
}


// Отримати список відео по id каналу
func getVideosByChannelId(id string, offset int) ([]byte, error) {
	Logger.Debugf("getVideosByChannelId(id: %v, offset: %v)", id, offset)

	if id == "" {
		return nil, errors.New("video id is null")
	}

	cacheId := id + "_" + strconv.Itoa(offset)

	// з кешем робимо тільки якщо він включений
	if *EnableCache {
		channeli, ok := cacheChannels.Get(cacheId)
		Logger.Debugf("id: %v, cache, have data? %v", cacheId, ok)

		// Перевіряємо чи є дані в кеші
		if ok {
			channel := channeli.(*YoutubeVideoShortInCache)
			Logger.Debugf("id: %v, timeUpdate: %v", cacheId, channel.timeUpdate)

			// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
			// ніж період перевірки списку відео
			if time.Since(channel.timeUpdate) < *PeriodVideoCache {
				Logger.Infof("id: %v, get videos for channel from cache", cacheId)
				Logger.Debugf("id: %v, cache, channel: %v", cacheId, string(channel.responce))

				return channel.responce, nil
			}
			Logger.Debugf("id: %v, cache, skip", cacheId)
		}

	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	stringVideos, err := getVideosByChannelIdFromDB(id, offset)
	if err != nil {
		return nil, err
	}

	// з кешем робимо тільки якщо він включений
	if *EnableCache {
		// Додаємо запит до кешу
		cacheChannels.Add(cacheId, &YoutubeVideoShortInCache{time.Now(), stringVideos})
		Logger.Debugf("id: %v, cache, add channels", cacheId)
	}

	Logger.Infof("id: %v, get videos for channel skip cache", cacheId)
	return stringVideos, nil
}

// Отримати список каналів
func getPlaylists(onlyEnable bool) ([]byte, error) {
	Logger.Debugf("getPlaylists(onlyEnable: %v)", onlyEnable)
	// з кешем робимо тільки якщо він включений та це не запит адміністратора на всі канали
	if *EnableCache && onlyEnable {
		Logger.Debugf("channels, cache, timeUpdate: %v", listCacheChannels.timeUpdate)
		
		// Дані з кешу беремо тільки якщо з останнього запиту пройшло часу менш
		// ніж період перевірки списку каналів
		if time.Since(listCacheChannels.timeUpdate) < *PeriodChannelCache {
			Logger.Debugf("channels, cache, list channels: %v", string(listCacheChannels.responce))
			Logger.Info("get channel from cache")

			return listCacheChannels.responce, nil
		}
		Logger.Debug("channels, cache, skip")
	}

	// В кеші актуальної інформации не знайдено, запрошуемо в БД
	response, err := getChannelsFromDB(onlyEnable)
	if err != nil {
		return nil, err
	}

	responceChannel := ResponseChannel{*MaxViewVideosInChannel, response}
	
	// Конвертуємо відповідь в json-формат
	stringJsonPlaylists, err := json.Marshal(&responceChannel)

	if err != nil {
		Logger.Errorf("Error convert select to Playlists: response=%v, error=%v", stringJsonPlaylists, err)
		return nil, err
	}

	Logger.Debugf("channels: %v", string(stringJsonPlaylists))

	// з кешем робимо тільки якщо він включений та це не запит адміністратора на всі канали
	if *EnableCache && onlyEnable {
		// Додаємо запит до кешу
		listCacheChannels.update(stringJsonPlaylists)
		Logger.Debugf("channels, cache, add list channels")
	}

	Logger.Info("get channel skip cache")
	return stringJsonPlaylists, nil
}

func getGlobalCounts(version string) ([]byte, error) {
	Logger.Debug("get globalCounts")
	if globalCounts == nil || time.Since(globalCounts.TimeUpdate) > *PeriodVideoCache {
		g, err := getGlobalCountsFromDB(version)
		if err == nil {
			globalCounts = g;
		}		
	} 		

	// Конвертуємо відповідь в json-формат
	stringGlobalCounts, err := json.Marshal(&globalCounts)

	if err != nil {
		Logger.Errorf("Error convert select to Playlists: response=%v, error=%v", globalCounts, err)
		return nil, err
	}
	
	Logger.Debugf("globalCounts: %v", string(stringGlobalCounts))
	
	return stringGlobalCounts, nil
}
