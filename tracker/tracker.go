package tracker

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/TeamTenuki/twiddler/db"
)

type Tracker struct {
	p    Poker
	r    Reporter
	live []Stream
}

func NewTracker(p Poker, r Reporter) *Tracker {
	return &Tracker{
		p:    p,
		r:    r,
		live: make([]Stream, 0),
	}
}

func (t *Tracker) Track(c context.Context) {
	go t.p.Poke()

	go func() {
		<-c.Done()
		t.p.Close()
	}()

	db := db.FromContext(c)

	for streams := range t.p.Source() {
		reportable := t.excludeKnown(streams)
		reportable = t.excludeReported(c, reportable)

		rooms, err := t.rooms(c)
		if err != nil {
			log.Printf("Failed to retrieve rooms: %s", err)
			continue
		}

		for _, s := range reportable {
			c, cancel := context.WithTimeout(c, 30*time.Second)
			defer cancel()

			tx := db.MustBeginTx(c, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})

			_, err := tx.Exec(`INSERT INTO [reports] ([user_id], [stream_id], [started_at])
			VALUES (?, ?, ?)`,
				s.UserID,
				s.ID,
				s.StartedAt)

			if err != nil {
				log.Printf("Failed to insert report: %s", err)
				tx.Rollback()
				continue
			}

			if err := t.report(c, rooms, s); err != nil {
				log.Printf("Failed to report stream: %s", err)
				tx.Rollback()
				continue
			}

			tx.Commit()
		}

		t.live = streams
	}
}

func (t *Tracker) rooms(c context.Context) (rooms []string, err error) {
	db := db.FromContext(c)

	err = db.SelectContext(c, &rooms, `SELECT [room_id] FROM [rooms]`)

	return rooms, err
}

func (t *Tracker) report(c context.Context, rs []string, s Stream) error {
	for _, r := range rs {
		if err := t.r.Report(c, r, &s); err != nil {
			return err
		}
	}

	return nil
}

func (t *Tracker) excludeKnown(ss []Stream) []Stream {
	unknown := make([]Stream, 0)

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

func (t *Tracker) excludeReported(c context.Context, ss []Stream) []Stream {
	reportable := make([]Stream, 0)

	for _, s := range ss {
		dt, err := t.lastReportTime(c, s)
		if err != nil {
			log.Printf("Failed to retrieve last report: %s", err)
		}

		if time.Since(dt) > time.Hour {
			reportable = append(reportable, s)
		}
	}

	return reportable
}

func (t *Tracker) lastReportTime(c context.Context, s Stream) (time.Time, error) {
	db := db.FromContext(c)

	// Default date/time that will be used if there are no rows for given channel.
	var reportTime = "2006-01-02T15:04:05Z"

	err := db.GetContext(c, &reportTime, `SELECT [started_at]
	FROM [reports]
	WHERE [user_id] = ?
	ORDER BY datetime([started_at]) DESC
	LIMIT 1`,
		s.UserID)

	if err != nil && err != sql.ErrNoRows {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, reportTime)
}
