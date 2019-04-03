package collector

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
			
	// Time elapsed since deleted video
	Duration time.Duration
}

func (video *YoutubeVideo) SetMetrics(CommentCount, LikeCount, DislikeCount, ViewCount uint64) {
	video.CommentCount = CommentCount
	video.LikeCount = LikeCount
	video.DislikeCount = DislikeCount
	video.ViewCount = ViewCount
	video.TimeCount = time.Now()
}

// Channel: A channel resource represents a YouTube channel.
type YoutubeChannel struct {
	Id string
	
	// Video: A list video resource represents a YouTube video.
	Videos map[string]*YoutubeVideo
	
	// is deleted or deactivated
	Deleted bool

	// Time elapsed since deleted Channel
	TimeDeleted time.Time
	
	Mux sync.Mutex
}

func (channel *YoutubeChannel) Append(id string, video *YoutubeVideo) {
	channel.Videos[id] = video	
}

// mark video as deleted from meter ( not in database )
func (channel *YoutubeChannel) SetDeletedVideo(id string) {
	channel.Videos[id].Deleted = true  	
	channel.Videos[id].TimeDeleted = time.Now()
}

func (channel *YoutubeChannel) Delete(id string) {
	delete(channel.Videos, id)	
}

type YoutubeChannels struct {
	// Словник посилань на канали
	Channels map[string]*YoutubeChannel
	Mux sync.Mutex
} 

// mark Channel as deleted from meter ( not in database )
func (channels *YoutubeChannels) SetDeletedChannel(id string) {
	channels.Channels[id].Deleted = true  	
	channels.Channels[id].TimeDeleted = time.Now()  	
}

// mark Channel as deleted from meter ( not in database )
func (channels *YoutubeChannels) CanselDeletedChannel(id string) {
	channels.Channels[id].Deleted = false  	
}

func (channels *YoutubeChannels) Delete(id string) {
	delete(channels.Channels, id)	
}

func (channels *YoutubeChannels) Append(id string) {	
	v := YoutubeChannel{Videos: make(map[string]*YoutubeVideo), Deleted: false, Id: id }
	channels.Channels[id] = &v  	
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
