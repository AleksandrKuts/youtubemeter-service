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
}
