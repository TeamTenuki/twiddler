package tracker

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/TeamTenuki/twiddler/db"
	"github.com/TeamTenuki/twiddler/messenger"
	"github.com/TeamTenuki/twiddler/stream"
	"github.com/TeamTenuki/twiddler/watcher"
)

type Tracker struct {
	w    watcher.Watcher
	m    messenger.Messenger
	live []stream.Stream
	err  error
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
		reportable := t.excludeKnown(streams)
		reportable = t.excludeReported(c, reportable)

		rooms, err := t.rooms(c)
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

		t.live = streams
	}
}

func (t *Tracker) Live() []stream.Stream {
	return t.live
}

func (t *Tracker) store(c context.Context, s *stream.Stream) {
	db := db.FromContext(c)

	_, t.err = db.ExecContext(c, `INSERT INTO [reports] ([user_id], [stream_id], [started_at])
VALUES (?, ?, ?)`,
		s.User.ID,
		s.ID,
		s.StartedAt.Format(time.RFC3339))
}

func (t *Tracker) rooms(c context.Context) (rooms []string, err error) {
	db := db.FromContext(c)

	err = db.SelectContext(c, &rooms, `SELECT [room_id] FROM [rooms]`)

	return rooms, err
}

func (t *Tracker) report(c context.Context, rs []string, s *stream.Stream) {
	if t.err != nil {
		return
	}

	for _, r := range rs {
		if err := t.m.MessageStream(c, r, s); err != nil {
			t.err = err

			break
		}
	}
}

func (t *Tracker) excludeKnown(ss []stream.Stream) []stream.Stream {
	unknown := make([]stream.Stream, 0)

outer:
	for _, s := range ss {
		for _, known := range t.live {
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
		if t.wasReported(c, s) {
			continue
		}

		dt, err := t.lastReportTimeForUser(c, &s)
		if err != nil {
			log.Printf("Failed to retrieve last report: %s", err)
		}

		if time.Since(dt) > time.Hour {
			reportable = append(reportable, s)
		}
	}

	return reportable
}

func (t *Tracker) wasReported(c context.Context, s stream.Stream) bool {
	db := db.FromContext(c)

	return sql.ErrNoRows != db.GetContext(c, new(string), `SELECT [started_at]
	FROM [reports]
	WHERE [stream_id] = ?
	LIMIT 1`,
		s.ID)
}

func (t *Tracker) lastReportTimeForUser(c context.Context, s *stream.Stream) (time.Time, error) {
	db := db.FromContext(c)

	// Default date/time that will be used if there are no rows for given channel.
	var reportTime = "2006-01-02T15:04:05Z"

	err := db.GetContext(c, &reportTime, `SELECT [started_at]
	FROM [reports]
	WHERE [user_id] = ?
	ORDER BY datetime([started_at]) DESC
	LIMIT 1`,
		s.User.ID)

	if err != nil && err != sql.ErrNoRows {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, reportTime)
}
