package twiddler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/TeamTenuki/twiddler/tracker"
)

type twitchPoker struct {
	d      time.Duration
	c      chan []tracker.Stream
	apiKey string
}

func newTwitchPoker(apiKey string, d time.Duration) tracker.Poker {
	return &twitchPoker{
		d:      d,
		c:      make(chan []tracker.Stream),
		apiKey: apiKey,
	}
}

func (p *twitchPoker) Poke(c context.Context) error {
	ticker := time.NewTicker(p.d)

	for {
		select {
		case <-c.Done():
			return p.Close()
		case <-ticker.C:
			p.poke(c)
		}
	}
}

func (p *twitchPoker) poke(c context.Context) {
	req, err := http.NewRequestWithContext(
		c,
		"GET",
		"https://api.twitch.tv/helix/streams?game_id=65360&first=100", nil)

	if err != nil {
		log.Panicf("Unexpected error: %s", err)
	}
	req.Header.Add("Client-ID", p.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if err != context.Canceled {
			log.Printf("Failed to perform HTTP request: %s", err)
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var streamContainer tracker.StreamContainer

		if err := json.NewDecoder(resp.Body).Decode(&streamContainer); err != nil {
			log.Printf("Failed to decode JSON: %s", err)
			return
		}

		p.c <- streamContainer.Data
	}
}

func (p *twitchPoker) Close() error {
	close(p.c)
	return nil
}

func (p *twitchPoker) Source() <-chan []tracker.Stream {
	return p.c
}
