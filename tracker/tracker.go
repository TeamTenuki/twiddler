package tracker

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/TeamTenuki/twiddler/clock"
	"github.com/TeamTenuki/twiddler/db"
	"github.com/TeamTenuki/twiddler/messenger"
	"github.com/TeamTenuki/twiddler/stream"
	"github.com/TeamTenuki/twiddler/watcher"
)

type Tracker struct {
	w      watcher.Watcher
	m      messenger.Messenger
	live   []stream.Stream
	liveMu sync.RWMutex
	err    error
}

func NewTracker(w watcher.Watcher, m messenger.Messenger) *Tracker {
	return &Tracker{
		w:    w,
		m:    m,
		live: make([]stream.Stream, 0),
	}
}

func (t *Tracker) Track(c context.Context) {
	go t.w.Watch(c)

	for streams := range t.w.Source() {
		// Update the info of the last time this stream was observed online.
		t.updateObservedAt(c, streams)

		reportable := t.excludeKnown(streams)
		reportable = t.excludeReported(c, reportable)
		reportable = t.excludeDuplicates(c, reportable)

		rooms, err := db.RoomsAll(c)
		if err != nil {
			log.Printf("Failed to retrieve rooms: %s", err)
			continue
		}

		for _, s := range reportable {
			t.store(c, &s)
			t.report(c, rooms, &s)

			if t.err != nil {
				log.Printf("Failed to report the stream: %s", t.err)
				t.err = nil
			}
		}

		t.setLive(streams)
	}
}

func (t *Tracker) Live() []stream.Stream {
	t.liveMu.RLock()
	defer t.liveMu.RUnlock()

	return t.live
}

func (t *Tracker) setLive(s []stream.Stream) {
	t.liveMu.Lock()
	t.live = s
	t.liveMu.Unlock()
}

func (t *Tracker) store(c context.Context, s *stream.Stream) {
	t.err = db.ReportStore(c, db.Report{
		UserID:     s.User.ID,
		StreamID:   s.ID,
		StartedAt:  s.StartedAt,
		ObservedAt: clock.NowUTC(),
	})
}

func (t *Tracker) updateObservedAt(c context.Context, ss []stream.Stream) {
	streamIDs := make([]string, len(ss))
	for i := range ss {
		streamIDs[i] = ss[i].ID
	}

	t.err = db.ReportObserveForStreams(c, streamIDs, clock.NowUTC())
}

func (t *Tracker) report(c context.Context, rs []db.Room, s *stream.Stream) {
	if t.err != nil {
		return
	}

	for _, r := range rs {
		if err := t.m.MessageStream(c, r.ID, s); err != nil {
			t.err = err

			break
		}
	}
}

func (t *Tracker) excludeKnown(ss []stream.Stream) []stream.Stream {
	unknown := make([]stream.Stream, 0)

outer:
	for _, s := range ss {
		for _, known := range t.Live() {
			if s.ID == known.ID {
				continue outer
			}
		}

		unknown = append(unknown, s)
	}

	return unknown
}

func (t *Tracker) excludeReported(c context.Context, ss []stream.Stream) []stream.Stream {
	reportable := make([]stream.Stream, 0)

	for _, s := range ss {
		// Do not report stream with the same stream ID twice.
		if yes, _ := db.ReportWasReported(c, s.ID); yes {
			continue
		}

		// If it is a stream restart (the stream ID has changed), check if
		// this user's latest stream report has happened less than an hour ago.

		dt, err := t.lastObservedTimeForUser(c, &s)
		if err != nil {
			log.Printf("Failed to retrieve last report: %s", err)
		}

		if clock.Since(dt) > time.Hour {
			reportable = append(reportable, s)
		} else {
			// Even though it isn't reportable, store it anyway, so it won't get
			// reported later.
			t.store(c, &s)
		}
	}

	return reportable
}

func (t *Tracker) excludeDuplicates(c context.Context, ss []stream.Stream) []stream.Stream {
	reportable := make([]stream.Stream, 0)
	seen := make(map[stream.Stream]struct{})

	for _, s := range ss {
		if _, yes := seen[s]; !yes {
			reportable = append(reportable, s)
			seen[s] = struct{}{}
		}
	}

	return reportable
}

func (t *Tracker) lastObservedTimeForUser(c context.Context, s *stream.Stream) (time.Time, error) {
	report, err := db.ReportLatestByUser(c, s.User.ID)

	switch {
	default:
		return report.ObservedAt, nil
	case err == sql.ErrNoRows:
		return time.Time{}, nil
	case err != nil:
		return time.Time{}, err
	}
}
