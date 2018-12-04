package server

import (
	"time"
)

// Channel: A channel resource contains information about a YouTube
type PlayList struct {
	// Id: The ID that YouTube uses to uniquely identify the playlist.
	Id string `json:"id"`

	// Title: The playlist's title.
	Title string `json:"title"`

	// Enable: if playlist is enabled
	Enable bool `json:"enable"`

	// Id: The ID that YouTube uses to uniquely identify the channel.
	Idch string `json:"idch"`
	
	// The date and time that the item was added to the list PlayLists
	Timeadd time.Time `json:"timeadd"`
	
	Countvideo int `json:"countvideo"`
}

type ResponcePlayList struct {
	MaxVideoCount  int `json:"maxvideocount"`
	PlayLists []PlayList `json:"playlists"`  
}

// Video: A video resource represents a YouTube video.
type YoutubeVideo struct {
	// PlaylistId: The ID that YouTube uses to uniquely identify the
	// playlist that the playlist item is in.
	PlaylistId string `json:"idpl"`

	// Title: The item's title.
	Title string `json:"title"`

	// Description: The item's description.
	Description string `json:"description"`


	// ChannelTitle: Channel title for the channel that the playlist item
	// belongs to.
	ChannelTitle string `json:"chtitle"`

	// ChannelId: The ID that YouTube uses to uniquely identify the user
	// that added the item to the playlist.
	ChannelId string `json:"chid"`

	// PublishedAt: The date and time that the item was added to the
	// playlist. The value is specified in ISO 8601 (YYYY-MM-DDThh:mm:ss.sZ)
	// format.
	PublishedAt time.Time `json:"publishedat"`

	CountMetrics int `json:"count"`

	MinTimeMetric time.Time `json:"mintime"`

	MaxTimeMetric time.Time `json:"maxtime"`	
}


// Video: A video resource represents a YouTube video.
type YoutubeVideoShort struct {
	//  VideoId: The ID that YouTube uses to uniquely identify the video
	Id string `json:"id"`

	// Title: The item's title.
	Title string `json:"title"`

	// PublishedAt: The date and time that the item was added to the
	// playlist. The value is specified in ISO 8601 (YYYY-MM-DDThh:mm:ss.sZ)
	// format.
	PublishedAt time.Time `json:"publishedat"`
	
	// Title: The playlist's title.
	Ptitle string `json:"ptitle"`
}

// Metric: A video resource represents a metric YouTube video.
type Metrics struct {
	// CommentCount: The number of comments for the video.
	CommentCount uint64 `json:"comment"`

	// LikeCount: The number of users who have indicated that they liked the
	// video by giving it a positive rating.
	LikeCount uint64 `json:"like"`

	// DislikeCount: The number of users who have indicated that they
	// disliked the video by giving it a negative rating.
	DislikeCount uint64 `json:"dislike"`

	// ViewCount: The number of times the video has been viewed.
	ViewCount uint64 `json:"view"`

	// Last poll time to get metrics
	Time time.Time `json:"mtime"`
}

// Структура для кешу списка плейлистів 
type ListPlayListInCache struct {
	// Час останнього запиту списку плейлистів
	timeUpdate time.Time  
	
	// Дані по відео 
	responce []byte
}

func (l *ListPlayListInCache) update( responce []byte) {
	l.timeUpdate = time.Now()
	l.responce = responce
}

func (l *ListPlayListInCache) reset() {
	l.timeUpdate = MIN_TIME
}

// Структура для кешу списку відео без плейлиста
type YoutubeVideoShortInCache struct {
	// Час останнього запиту списку відео
	timeUpdate time.Time  
	
	// Дані по відео 
	responce []byte
}


// Структура для кешу метрик та опису відео
type VideoInCache struct {
	// Час останнього запиту метрик. Данні за цей період не змінюються (максимум додасться одна метрика)
	updateMetrics time.Time  

	// Час останнього запиту статистичних даних по відео (включає дані по метрикам див. YoutubeVideo struct). 
	// Данні за цей період не змінюються (максимум додасться одна метрика)
	updateVideo time.Time
	
	// Час додавання відео в плейлист. Поза періодом збору метрик, який рахується з даного часу дані вже не міняються
	// тому беруться тільки з кешу
	publishedAt time.Time
	
	// Дані по відео 
	videoResponce []byte
	
	// Метрики
	metricsResponce []byte
}

func (v *VideoInCache) updateCacheVideo(publishedAt time.Time, videoResponce []byte) {
	v.updateVideo = time.Now()
	v.publishedAt = publishedAt
	v.videoResponce = videoResponce
}

func (v *VideoInCache) updateCacheMetrics(metricsResponce []byte) {
	v.updateMetrics = time.Now()
	v.metricsResponce = metricsResponce
}

// Структура для кешу списку відео без плейлиста
type GlobalCounts struct {
	// Час останнього запиту списку відео
	TimeUpdate time.Time `json:"timeupdate"`
	
	// Кількість плейлистів 
	CountPlaylists int `json:"countpl"`
	
	// Кількість відео 
	CountVideos int `json:"countvideo"`
	
	MaxVideoCount  int `json:"maxcountvideo"`
	
	PeriodVideoCache time.Duration `json:"periodvideocache"`	
	
	Version string `json:"version"`
	
	ListenAdmin  bool `json:"listenadmin"`
}

