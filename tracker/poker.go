package tracker

import (
	"context"
)

type Poker interface {
	Poke(c context.Context) error
	Close() error
	Source() <-chan []Stream
}

type Reporter interface {
	Report(c context.Context, roomID string, s *Stream) error
	ReportMessage(c context.Context, roomID string, content string) error
}

// Stream describes relevant information about a Twitch channel.
type Stream struct {
	// Twitch username of the channel owner.
	UserName string `json:"user_name"`

	// Twitch user ID.
	UserID string `json:"user_id"`

	// Channel title.
	Title string `json:"title"`

	// Live stream thumbnail URL.
	Thumbnail string `json:"thumbnail_url"`

	// Unique stream identifier.
	ID string `json:"id" db:"stream_id"`

	// ISO-8601 date/time of stream going live.
	StartedAt string `json:"started_at" db:"started_at"`
}

// StreamContainer represents unmarshalled JSON response from Twitch.
type StreamContainer struct {
	Data       []Stream `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor,omitempty"`
	} `json:"pagination,omitempty"`
}
