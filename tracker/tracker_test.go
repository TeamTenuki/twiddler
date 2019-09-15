package tracker_test

import (
	"context"
	"testing"
	"time"

	"github.com/TeamTenuki/twiddler/db"
	"github.com/TeamTenuki/twiddler/stream"
	"github.com/TeamTenuki/twiddler/testutil"
)

func TestStreamIsReported(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	baselineTime := time.Now().UTC()

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime,
		},
	})

	tr.Close()
	tr.Wait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1")
}

func TestSameStreamTwiceInOneBatchReportedOnce(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	baselineTime := time.Now().UTC()

	tr.Send([]stream.Stream{
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

	tr.Close()
	tr.Wait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1")
}

func TestInterleavedStreamReportIsReportedOnlyOnce(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	baselineTime := time.Now().UTC()

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime,
		},
	})

	tr.Send([]stream.Stream{ /* empty */ })
	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime,
		},
	})

	tr.Close()
	tr.Wait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1")
}

func TestStreamRestartsWithinOneHourSingleReport(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	baselineTime := time.Now().UTC()

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime.Add(-10 * time.Minute),
		},
	})

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream2",
			StartedAt: baselineTime,
		},
	})

	tr.Close()
	tr.Wait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1")
}

func TestStreamRestartsMoreThanHourGapBothReported(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	baselineTime := time.Now().UTC()

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime.Add(-1*time.Hour - 10*time.Minute),
		},
	})

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream2",
			StartedAt: baselineTime,
		},
	})

	tr.Close()
	tr.Wait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1", "stream2")
}

func TestSameStreamOneHourLaterNotReported(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	baselineTime := time.Now().UTC()

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime.Add(-1*time.Hour - 10*time.Minute),
		},
	})

	tr.Send([]stream.Stream{})

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime,
		},
	})

	tr.Close()
	tr.Wait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1")
}

//
// HELPERS
//
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
// DB
//
func setupDB(c context.Context) {
	db := db.FromContext(c)
	db.MustExec(`INSERT INTO [rooms] ([room_id]) VALUES ('room1')`)
}
