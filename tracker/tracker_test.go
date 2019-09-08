package tracker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/TeamTenuki/twiddler/db"
	"github.com/TeamTenuki/twiddler/tracker"
)

func TestStreamIsReported(t *testing.T) {
	p, r, wg := setupTracker()
	baselineTime := time.Now()

	p.poke([]tracker.Stream{
		{
			UserID:    "user1",
			ID:        "stream1",
			StartedAt: baselineTime.Format(time.RFC3339),
		},
	})

	p.Close()
	wg.Wait()

	bag := r.rooms["room1"]
	expectStreamReports(t, bag.streams, "stream1")
}

func TestInterleavedStreamReportIsReportedOnlyOnce(t *testing.T) {
	p, r, wg := setupTracker()
	baselineTime := time.Now()

	p.poke([]tracker.Stream{
		{
			UserID:    "user1",
			ID:        "stream1",
			StartedAt: baselineTime.Format(time.RFC3339),
		},
	})

	p.poke([]tracker.Stream{ /* empty */ })
	p.poke([]tracker.Stream{
		{
			UserID:    "user1",
			ID:        "stream1",
			StartedAt: baselineTime.Format(time.RFC3339),
		},
	})

	p.Close()
	wg.Wait()

	bag := r.rooms["room1"]
	expectStreamReports(t, bag.streams, "stream1")
}

func TestStreamRestartsWithinOneHourSingleReport(t *testing.T) {
	p, r, wg := setupTracker()
	baselineTime := time.Now()

	p.poke([]tracker.Stream{
		{
			UserID:    "user1",
			ID:        "stream1",
			StartedAt: baselineTime.Add(-10 * time.Minute).Format(time.RFC3339),
		},
	})

	p.poke([]tracker.Stream{
		{
			UserID:    "user1",
			ID:        "stream2",
			StartedAt: baselineTime.Format(time.RFC3339),
		},
	})

	p.Close()
	wg.Wait()

	bag := r.rooms["room1"]
	expectStreamReports(t, bag.streams, "stream1")
}

func TestStreamRestartsMoreThanHourGapBothReported(t *testing.T) {
	p, r, wg := setupTracker()
	baselineTime := time.Now()

	p.poke([]tracker.Stream{
		{
			UserID:    "user1",
			ID:        "stream1",
			StartedAt: baselineTime.Add(-2 * time.Hour).Format(time.RFC3339),
		},
	})

	p.poke([]tracker.Stream{
		{
			UserID:    "user1",
			ID:        "stream2",
			StartedAt: baselineTime.Format(time.RFC3339),
		},
	})

	p.Close()
	wg.Wait()

	bag := r.rooms["room1"]
	expectStreamReports(t, bag.streams, "stream1", "stream2")
}

//
// HELPERS
//

func setupTracker() (*pokerT, *reporterT, *sync.WaitGroup) {
	var wg sync.WaitGroup

	wg.Add(1) // We expect single report.

	p := newPoker()
	r := newReporter()
	c := setupDB()

	tracker := tracker.NewTracker(p, r)
	go func() {
		tracker.Track(c)
		wg.Done()
	}()

	return p, r, &wg
}

func expectStreamReports(t *testing.T, ss []*tracker.Stream, ids ...string) {
	if len(ids) != len(ss) {
		t.Errorf("Expected %d reports got %d", len(ids), len(ss))
	}

	for _, s := range ss {
		exists := false
		for _, id := range ids {
			exists = exists || s.ID == id
		}

		if !exists {
			t.Errorf("Expected report for stream ID %q", s.ID)
		}
	}
}

//
// REPORTER
//

type bagT struct {
	streams  []*tracker.Stream
	messages []string
}

type reporterT struct {
	rooms map[string]bagT
}

func newReporter() *reporterT {
	return &reporterT{
		rooms: make(map[string]bagT),
	}
}

func (r *reporterT) Report(c context.Context, roomID string, s *tracker.Stream) error {
	bag := r.rooms[roomID]
	bag.streams = append(bag.streams, s)

	r.rooms[roomID] = bag

	return nil
}

func (r *reporterT) ReportMessage(c context.Context, roomID string, content string) error {
	bag := r.rooms[roomID]
	bag.messages = append(bag.messages, content)

	r.rooms[roomID] = bag

	return nil
}

//
// POKER
//

type pokerT struct {
	c chan []tracker.Stream
}

func newPoker() *pokerT {
	return &pokerT{
		c: make(chan []tracker.Stream, 1),
	}
}

func (p *pokerT) poke(ss []tracker.Stream) {
	p.c <- ss
}

func (p *pokerT) Poke() error {
	return nil
}

func (p *pokerT) Close() error {
	close(p.c)

	return nil
}

func (p *pokerT) Source() chan []tracker.Stream {
	return p.c
}

//
// SETUP DB
//
func setupDB() context.Context {
	db.MustInit(":memory:")

	c := db.NewContext(context.Background())
	db.SetupDB(c)
	db := db.FromContext(c)

	db.MustExec(`INSERT INTO [rooms] ([room_id]) VALUES ('room1')`)

	return c
}
