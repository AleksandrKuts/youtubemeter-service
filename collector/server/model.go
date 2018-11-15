package server

import (
	"sync"
	"time"
)

// Video: A video resource represents a YouTube video.
type YoutubeVideo struct {
	// PublishedAt: The date and time that the video was uploaded. The value
	// is specified in ISO 8601 (YYYY-MM-DDThh:mm:ss.sZ) format.
	PublishedAt time.Time `json:"publishedAt,omitempty"`

	// Title: The video's title.
	Title string `json:"title,omitempty"`
	
	// CommentCount: The number of comments for the video.
	CommentCount uint64 `json:"commentCount,omitempty,string"`

	// DislikeCount: The number of users who have indicated that they
	// disliked the video by giving it a negative rating.
	DislikeCount uint64 `json:"dislikeCount,omitempty,string"`

	// LikeCount: The number of users who have indicated that they liked the
	// video by giving it a positive rating.
	LikeCount uint64 `json:"likeCount,omitempty,string"`

	// ViewCount: The number of times the video has been viewed.
	ViewCount uint64 `json:"viewCount,omitempty,string"`
	
	// Last poll time to get metrics
	timeCount time.Time
	
	// is deleted or deactivated
	deleted bool
	
	// Time elapsed since deleted video
	timeDeleted time.Time
			
}

func (video *YoutubeVideo) setMetrics(CommentCount, LikeCount, DislikeCount, ViewCount uint64) {
	video.CommentCount = CommentCount
	video.LikeCount = LikeCount
	video.DislikeCount = DislikeCount
	video.ViewCount = ViewCount
	video.timeCount = time.Now()
}


type YoutubePlayList struct {
	id string
	
	// Video: A list video resource represents a YouTube video.
	videos map[string]*YoutubeVideo
	
	// is deleted or deactivated
	deleted bool

	// Time elapsed since deleted PlayList
	timeDeleted time.Time
	
	mux sync.Mutex
}

func (playlist *YoutubePlayList) append(id string, video *YoutubeVideo) {
	playlist.videos[id] = video	
}

// mark playlist as deleted from meter ( not in database )
func (playlist *YoutubePlayList) setDeletedVideo(id string) {
	playlist.videos[id].deleted = true  	
	playlist.videos[id].timeDeleted = time.Now()
}

func (playlist *YoutubePlayList) delete(id string) {
	delete(playlist.videos, id)	
}

type YoutubePlayLists struct {
	// Словник посилань на ПлейЛисти
	playlists map[string]*YoutubePlayList
	mux sync.Mutex
} 

// mark playlist as deleted from meter ( not in database )
func (playlists *YoutubePlayLists) setDeletedPlayList(id string) {
	playlists.playlists[id].deleted = true  	
	playlists.playlists[id].timeDeleted = time.Now()  	
}

// mark playlist as deleted from meter ( not in database )
func (playlists *YoutubePlayLists) canselDeletedPlayList(id string) {
	playlists.playlists[id].deleted = false  	
}

func (playlists *YoutubePlayLists) delete(id string) {
	delete(playlists.playlists, id)	
}

func (playlists *YoutubePlayLists) append(id string) {	
	v := YoutubePlayList{videos: make(map[string]*YoutubeVideo), deleted: false, id: id }
	playlists.playlists[id] = &v  	
}

// Список плейлистів для збору статистики. Список корегується згідно з розкладом (config.PeriodPlayList) 
var playlists YoutubePlayLists = YoutubePlayLists{playlists: make(map[string]*YoutubePlayList)}
