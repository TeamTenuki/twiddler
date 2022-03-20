package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/twitch"

	"github.com/TeamTenuki/twiddler/clock"
	"github.com/TeamTenuki/twiddler/stream"
)

var _ stream.Fetcher = &Fetcher{}

type Fetcher struct {
	oauth2Config clientcredentials.Config
	tokenSource  oauth2.TokenSource
	r            *strings.Replacer
	userCache    map[string]entryT
}

func NewFetcher(c context.Context, clientID string, clientSecret string) *Fetcher {
	oauth2Config := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     twitch.Endpoint.TokenURL,
	}

	return &Fetcher{
		oauth2Config: oauth2Config,
		tokenSource:  oauth2Config.TokenSource(c),
		r:            strings.NewReplacer("{width}", "1280", "{height}", "720"),
		userCache:    make(map[string]entryT),
	}
}

func (f *Fetcher) Fetch(c context.Context) ([]stream.Stream, error) {
	var streamContainer streamContainerT

	err := f.get(c, "https://api.twitch.tv/helix/streams?game_id=65360&first=100", &streamContainer)
	if err != nil {
		return nil, err
	}

	return f.constructStreamList(c, &streamContainer)
}

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

	user, err := f.userInfo(c, s.UserID)
	if err != nil {
		return stream.Stream{}, err
	}

	cs := stream.Stream{
		ID:           s.ID,
		User:         user,
		Title:        s.Title,
		StartedAt:    startedAt.In(time.UTC),
		ThumbnailURL: thumbnailURL,
	}

	return cs, nil
}

func (f *Fetcher) userInfo(c context.Context, userID string) (stream.User, error) {
	if entry, exists := f.userCache[userID]; exists && !entry.expired() {
		return entry.user, nil
	}

	var userContainer userContainerT

	err := f.get(c, "https://api.twitch.tv/helix/users?id="+userID, &userContainer)
	if err != nil {
		return stream.User{}, err
	}

	if len(userContainer.Data) == 0 {
		return stream.User{}, fmt.Errorf("failed to retrieve user with ID %s", userID)
	}

	uc := userContainer.Data[0]

	channelURL, err := url.Parse(fmt.Sprintf("https://twitch.tv/%s", uc.Login))
	if err != nil {
		return stream.User{}, err
	}
	pictureURL, err := url.Parse(uc.ProfileImageURL)
	if err != nil {
		return stream.User{}, err
	}
	offlineImageUrl, err := url.Parse(uc.OfflineImageURL)
	if err != nil {
		return stream.User{}, err
	}

	u := stream.User{
		ID:              uc.ID,
		Name:            uc.Login,
		DisplayName:     uc.DisplayName,
		ChannelURL:      channelURL,
		ProfileURL:      channelURL,
		PictureURL:      pictureURL,
		OfflineImageURL: offlineImageUrl,
	}

	f.userCache[userID] = newEntry(u)

	return u, nil
}

func (f *Fetcher) get(c context.Context, u string, d any) error {
	req, err := f.newRequest(c, u)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return json.NewDecoder(resp.Body).Decode(d)
	}

	return fmt.Errorf("failed to fetch data, server replied with status %s", resp.Status)
}

func (f *Fetcher) newRequest(c context.Context, u string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(
		c,
		"GET",
		u,
		nil)
	if err != nil {
		return nil, err
	}

	token, err := f.tokenSource.Token()
	if err != nil {
		return nil, err
	}

	req.Header.Add("Client-ID", f.oauth2Config.ClientID)
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)

	return req, nil
}

type entryT struct {
	user      stream.User
	expiresAt time.Time
}

func newEntry(u stream.User) entryT {
	return entryT{user: u, expiresAt: clock.NowUTC().Add(48 * time.Hour)}
}

func (e *entryT) expired() bool {
	return e.expiresAt.After(clock.NowUTC())
}

type userContainerT struct {
	Data []userT `json:"data"`
}

type userT struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	ProfileImageURL string `json:"profile_image_url"`
	OfflineImageURL string `json:"offline_image_url"`
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
