package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/TeamTenuki/twiddler/stream"
)

type Fetcher struct {
	apiKey string
	r      *strings.Replacer
}

func NewFetcher(apiKey string) *Fetcher {
	return &Fetcher{
		apiKey: apiKey,
		r:      strings.NewReplacer("{width}", "1280", "{height}", "720"),
	}
}

func (f *Fetcher) Fetch(c context.Context) ([]stream.Stream, error) {
	req, err := http.NewRequestWithContext(
		c,
		"GET",
		"https://api.twitch.tv/helix/streams?game_id=65360&first=100",
		nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Client-ID", f.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var streamContainer streamContainerT

		if err := json.NewDecoder(resp.Body).Decode(&streamContainer); err != nil {
			return nil, err
		}

		return f.constructStreamList(c, &streamContainer)
	}

	return nil, fmt.Errorf("failed to fetch data, server replied with status %s", resp.Status)
}

var _ stream.Fetcher = &Fetcher{}

func (f *Fetcher) constructStreamList(c context.Context, sc *streamContainerT) ([]stream.Stream, error) {
	ss := make([]stream.Stream, len(sc.Data))

	for i := range sc.Data {
		s, err := f.constructStream(c, &sc.Data[i])
		if err != nil {
			return nil, err
		}
		ss[i] = s
	}

	return ss, nil
}

func (f *Fetcher) constructStream(c context.Context, s *streamT) (stream.Stream, error) {
	startedAt, err := time.Parse(time.RFC3339, s.StartedAt)
	if err != nil {
		return stream.Stream{}, err
	}

	thumbnailURL, err := url.Parse(f.r.Replace(s.Thumbnail))
	if err != nil {
		return stream.Stream{}, err
	}

	cs := stream.Stream{
		ID: s.ID,
		User: stream.User{
			ID:   s.UserID,
			Name: s.UserName,
		},
		Title:        s.Title,
		StartedAt:    startedAt.In(time.UTC),
		ThumbnailURL: thumbnailURL,
	}

	return cs, nil
}

type streamContainerT struct {
	Data       []streamT   `json:"data"`
	Pagination paginationT `json:"pagination"`
}

type streamT struct {
	// Unique stream identifier.
	ID string `json:"id"`

	// Twitch username of the channel owner.
	UserName string `json:"user_name"`

	// Twitch user ID.
	UserID string `json:"user_id"`

	// Channel title.
	Title string `json:"title"`

	// Live stream thumbnail URL.
	Thumbnail string `json:"thumbnail_url"`

	// ISO-8601 date/time of stream going live.
	StartedAt string `json:"started_at"`
}

type paginationT struct {
	Cursor string `json:"cursor"`
}
