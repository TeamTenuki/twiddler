package watcher

import (
	"context"
	"log"
	"time"

	"github.com/TeamTenuki/twiddler/stream"
)

// Watcher watches streaming service for live streams.
type Watcher interface {
	// Watch starts a watcher's loop. This has to be called prior to receiving any values
	// from Source.
	Watch(c context.Context) error

	// Source returns a channel of stream lists.
	// In order to receive values on this channel, Watch has to be called before.
	Source() <-chan []stream.Stream
}

// Periodic returns a periodic watcher that fetches data from service with given fetcher
// periodically, with given duration.
func Periodic(f stream.Fetcher, d time.Duration) Watcher {
	return &periodicT{
		f: f,
		d: d,
		c: make(chan []stream.Stream),
	}
}

type periodicT struct {
	f stream.Fetcher
	d time.Duration
	c chan []stream.Stream
}

func (p *periodicT) Watch(c context.Context) error {
	ticker := time.NewTicker(p.d)
	for range ticker.C {
		select {
		default:
			p.check(c)
		case <-c.Done():
			ticker.Stop()

			return nil
		}
	}
	return nil
}

func (p *periodicT) Source() <-chan []stream.Stream {
	return p.c
}

func (p *periodicT) check(c context.Context) {
	streamList, err := p.fetch(c)
	if err != nil {
		if err != context.Canceled {
			log.Printf("Failed to fetch stream list: %s", err)
		}

		return
	}

	p.c <- streamList
}

func (p *periodicT) fetch(c context.Context) ([]stream.Stream, error) {
	ss, err := p.f.Fetch(c)
	if err != nil {
		return nil, err
	}

	return ss, nil
}
