package model

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
	TimeCount time.Time
	
	// is deleted or deactivated
	Deleted bool
	
	// Time elapsed since deleted video
	TimeDeleted time.Time
			
}

func (video *YoutubeVideo) SetMetrics(CommentCount, LikeCount, DislikeCount, ViewCount uint64) {
	video.CommentCount = CommentCount
	video.LikeCount = LikeCount
	video.DislikeCount = DislikeCount
	video.ViewCount = ViewCount
	video.TimeCount = time.Now()
}

type YoutubePlayList struct {
	Id string
	
	// Video: A list video resource represents a YouTube video.
	Videos map[string]*YoutubeVideo
	
	// is deleted or deactivated
	Deleted bool

	// Time elapsed since deleted PlayList
	TimeDeleted time.Time
	
	Mux sync.Mutex
}

func (playlist *YoutubePlayList) Append(id string, video *YoutubeVideo) {
	playlist.Videos[id] = video	
}

// mark playlist as deleted from meter ( not in database )
func (playlist *YoutubePlayList) SetDeletedVideo(id string) {
	playlist.Videos[id].Deleted = true  	
	playlist.Videos[id].TimeDeleted = time.Now()
}

func (playlist *YoutubePlayList) Delete(id string) {
	delete(playlist.Videos, id)	
}

type YoutubePlayLists struct {
	// Словник посилань на ПлейЛисти
	Playlists map[string]*YoutubePlayList
	Mux sync.Mutex
} 

// mark playlist as deleted from meter ( not in database )
func (playlists *YoutubePlayLists) SetDeletedPlayList(id string) {
	playlists.Playlists[id].Deleted = true  	
	playlists.Playlists[id].TimeDeleted = time.Now()  	
}

// mark playlist as deleted from meter ( not in database )
func (playlists *YoutubePlayLists) CanselDeletedPlayList(id string) {
	playlists.Playlists[id].Deleted = false  	
}

func (playlists *YoutubePlayLists) Delete(id string) {
	delete(playlists.Playlists, id)	
}

func (playlists *YoutubePlayLists) Append(id string) {	
	v := YoutubePlayList{Videos: make(map[string]*YoutubeVideo), Deleted: false, Id: id }
	playlists.Playlists[id] = &v  	
}

// Video: A video resource represents a YouTube video.
type Metrics struct {
	//The id parameter specifies a comma-separated list of the YouTube video ID(s)
	Id string `json:"id"`

	// CommentCount: The number of comments for the video.
	CommentCount uint64 `json:"commentCount,omitempty,string"`

	// LikeCount: The number of users who have indicated that they liked the
	// video by giving it a positive rating.
	LikeCount uint64 `json:"likeCount,omitempty,string"`

	// DislikeCount: The number of users who have indicated that they
	// disliked the video by giving it a negative rating.
	DislikeCount uint64 `json:"dislikeCount,omitempty,string"`

	// ViewCount: The number of times the video has been viewed.
	ViewCount uint64 `json:"viewCount,omitempty,string"`

	// Last poll time to get metrics
	Time time.Time
}


//var playlists YoutubePlayLists = YoutubePlayLists{playlists: make(map[string]*YoutubePlayList)}
