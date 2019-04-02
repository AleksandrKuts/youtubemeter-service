package backend

import (
	"time"
)

// Channel: A channel resource contains information about a YouTube
type Channel struct {
	// Id: The ID that YouTube uses to uniquely identify the channel.
	Id string `json:"id"`

	// Title: The Channels's title.
	Title string `json:"title"`

	// Enable: if channel is enabled
	Enable bool `json:"enable"`

	// The date and time that the item was added
	Timeadd time.Time `json:"timeadd"`
	
	// count videos
	Countvideo int `json:"countvideo"`
}

type ResponseChannel struct {
	MaxVideoCount  int `json:"maxvideocount"`
	Channels []Channel `json:"channels"`  
}

// Video: A video resource represents a YouTube video.
type YoutubeVideo struct {
	// Title: The item's title.
	Title string `json:"title"`

	// Description: The item's description.
	Description string `json:"description"`

	// ChannelId: The ID that YouTube uses to uniquely identify the user
	// that added the item
	ChannelId string `json:"idch"`

	// ChannelTitle: Channel title for the channel
	ChannelTitle string `json:"chtitle"`

	// PublishedAt: The date and time that the item was added. 
	// The value is specified in ISO 8601 (YYYY-MM-DDThh:mm:ss.sZ)
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

	// PublishedAt: The date and time that the item was added.
	// The value is specified in ISO 8601 (YYYY-MM-DDThh:mm:ss.sZ)
	// format.
	PublishedAt time.Time `json:"publishedat"`
	
	// Chanel: The chanel's title.
	Ptitle string `json:"ptitle"`

	// Chanel: The chanel's title.
	Duration time.Duration `json:"duration"`
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

// Структура для кешу каналів
type ListChannelInCache struct {
	// Час останнього запиту списку каналів
	timeUpdate time.Time  
	
	// Дані по відео 
	responce []byte
}

func (l *ListChannelInCache) update( responce []byte) {
	l.timeUpdate = time.Now()
	l.responce = responce
}

func (l *ListChannelInCache) reset() {
	l.timeUpdate = MIN_TIME
}

// Структура для кешу списку відео
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
	
	// Час додавання відео. Поза періодом збору метрик, який рахується з даного часу дані вже не міняються
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

// Структура для кешу списку відео
type GlobalCounts struct {
	// Час останнього запиту списку відео
	TimeUpdate time.Time `json:"timeupdate"`
	
	// Кількість каналів
	CountChannels int `json:"countch"`
	
	// Кількість відео 
	CountVideos int `json:"countvideo"`
	
	MaxVideoCount  int `json:"maxcountvideo"`
	
	PeriodVideoCache time.Duration `json:"periodvideocache"`	
	
	Version string `json:"version"`
	
	ListenAdmin  bool `json:"listenadmin"`
}

