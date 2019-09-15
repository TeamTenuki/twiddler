package tracker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/TeamTenuki/twiddler/db"
	"github.com/TeamTenuki/twiddler/messenger"
	"github.com/TeamTenuki/twiddler/stream"
	"github.com/TeamTenuki/twiddler/tracker"
	"github.com/TeamTenuki/twiddler/watcher"
)

func TestStreamIsReported(t *testing.T) {
	tr := setupTracker()
	baselineTime := time.Now().UTC()

	tr.poke([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime,
		},
	})

	tr.close()
	tr.wait()

	logReports(t, tr.c)

	bag := tr.room("room1")
	expectStreamReports(t, bag.streams, "stream1")
}

func TestSameStreamTwiceInOneBatchReportedOnce(t *testing.T) {
	tr := setupTracker()
	baselineTime := time.Now().UTC()

	tr.poke([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime,
		},
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime,
		},
	})

	tr.close()
	tr.wait()

	logReports(t, tr.c)

	bag := tr.room("room1")
	expectStreamReports(t, bag.streams, "stream1")
}

func TestInterleavedStreamReportIsReportedOnlyOnce(t *testing.T) {
	tr := setupTracker()
	baselineTime := time.Now().UTC()

	tr.poke([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime,
		},
	})

	tr.poke([]stream.Stream{ /* empty */ })
	tr.poke([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime,
		},
	})

	tr.close()
	tr.wait()

	logReports(t, tr.c)

	bag := tr.room("room1")
	expectStreamReports(t, bag.streams, "stream1")
}

func TestStreamRestartsWithinOneHourSingleReport(t *testing.T) {
	tr := setupTracker()
	baselineTime := time.Now().UTC()

	tr.poke([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime.Add(-10 * time.Minute),
		},
	})

	tr.poke([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream2",
			StartedAt: baselineTime,
		},
	})

	tr.close()
	tr.wait()

	logReports(t, tr.c)

	bag := tr.room("room1")
	expectStreamReports(t, bag.streams, "stream1")
}

func TestStreamRestartsMoreThanHourGapBothReported(t *testing.T) {
	tr := setupTracker()
	baselineTime := time.Now().UTC()

	tr.poke([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime.Add(-1*time.Hour - 10*time.Minute),
		},
	})

	tr.poke([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream2",
			StartedAt: baselineTime,
		},
	})

	tr.close()
	tr.wait()

	logReports(t, tr.c)

	bag := tr.room("room1")
	expectStreamReports(t, bag.streams, "stream1", "stream2")
}

func TestSameStreamOneHourLaterNotReported(t *testing.T) {
	tr := setupTracker()
	baselineTime := time.Now().UTC()

	tr.poke([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime.Add(-1*time.Hour - 10*time.Minute),
		},
	})

	tr.poke([]stream.Stream{})

	tr.poke([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime,
		},
	})

	tr.close()
	tr.wait()

	logReports(t, tr.c)

	bag := tr.room("room1")
	expectStreamReports(t, bag.streams, "stream1")
}

//
// HELPERS
//

type trackerT struct {
	c  context.Context
	w  *watcherT
	m  *messengerT
	wg *sync.WaitGroup
}

func setupTracker() *trackerT {
	wg := &sync.WaitGroup{}
	wg.Add(1) // We expect single report.

	w := newWatcher()
	m := newMessenger()
	c := setupDB()

	tracker := tracker.NewTracker(w, m)
	go func() {
		tracker.Track(c)
		wg.Done()
	}()

	return &trackerT{
		c:  c,
		w:  w,
		m:  m,
		wg: wg,
	}
}

func (t *trackerT) poke(ss []stream.Stream) {
	t.w.poke(ss)
}

func (t *trackerT) close() {
	t.w.close()
}

func (t *trackerT) wait() {
	t.wg.Wait()
}

func (t *trackerT) room(roomID string) bagT {
	return t.m.rooms[roomID]
}

func expectStreamReports(t *testing.T, ss []*stream.Stream, ids ...string) {
	t.Helper()

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
	streams  []*stream.Stream
	messages []string
}

var _ messenger.Messenger = &messengerT{}

type messengerT struct {
	rooms map[string]bagT
}

func newMessenger() *messengerT {
	return &messengerT{
		rooms: make(map[string]bagT),
	}
}

func (r *messengerT) MessageStream(c context.Context, roomID string, s *stream.Stream) error {
	bag := r.rooms[roomID]
	bag.streams = append(bag.streams, s)

	r.rooms[roomID] = bag

	return nil
}

func (r *messengerT) MessageStreamList(c context.Context, roomID string, ss []stream.Stream) error {
	return nil
}

func (r *messengerT) MessageText(c context.Context, roomID string, content string) error {
	bag := r.rooms[roomID]
	bag.messages = append(bag.messages, content)

	r.rooms[roomID] = bag

	return nil
}

//
// WATCHER
//

var _ watcher.Watcher = &watcherT{}

type watcherT struct {
	c chan []stream.Stream
}

func newWatcher() *watcherT {
	return &watcherT{
		c: make(chan []stream.Stream, 1),
	}
}

func (p *watcherT) Watch(c context.Context) error {
	return nil
}

func (p *watcherT) Source() <-chan []stream.Stream {
	return p.c
}

func (p *watcherT) poke(ss []stream.Stream) {
	p.c <- ss
}

func (p *watcherT) close() error {
	close(p.c)

	return nil
}

//
// DB
//
func setupDB() context.Context {
	db.MustInit(":memory:")

	c := db.NewContext(context.Background())
	db.SetupDB(c)
	db := db.FromContext(c)

	db.MustExec(`INSERT INTO [rooms] ([room_id]) VALUES ('room1')`)

	return c
}

type report struct {
	StreamID  string `db:"stream_id"`
	UserID    string `db:"user_id"`
	StartedAt string `db:"started_at"`
}

func logReports(t *testing.T, c context.Context) {
	t.Helper()

	db := db.FromContext(c)

	var reports []report
	err := db.Select(&reports, `SELECT [stream_id], [user_id], [started_at] FROM [reports]`)
	if err != nil {
		t.Errorf("Failed to retrieve reports: %s", err)
	}

	for i := range reports {
		t.Logf("%#v", reports[i])
	}
}
