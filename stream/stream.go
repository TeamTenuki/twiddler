package stream

import (
	"context"
	"net/url"
	"time"
)

// Stream rerpresents a generic stream information. Depending on the service, not all fields
// may be filled with reasonable values.
type Stream struct {
	// ID is a unique identifier of this stream on a given service.
	ID string `db:"stream_id"`

	// User is an information about streamer of this stream on a given service.
	User User

	// Title of this stream.
	Title string

	// ThumbnailURL of this stream.
	ThumbnailURL *url.URL

	// StartedAt is a date/time of this stream going live.
	StartedAt time.Time
}

// User representse a generic streamer's profile information. Depending on the service, not all
// fields may be filled with reasonable values.
type User struct {
	// ID is a unique identifier of a user on a given service.
	ID string

	// Name is a user's name on a given service.
	Name string

	// ChannelURL is an URL that points to streamer's channel page on a given service.
	ChannelURL *url.URL

	// ProfileURL is an URL that points to streamer's profile page on a given service.
	ProfileURL *url.URL

	// PictureURL is an URL that points to streamer's profile picture on a given service.
	PictureURL *url.URL
}

// Fetcher knows how to interact with streaming service to obtain stream infos.
type Fetcher interface {
	// Fetch fetches currently live streams info from a streaming service.
	// Implementation should honour context cancellation.
	Fetch(c context.Context) ([]Stream, error)
}
