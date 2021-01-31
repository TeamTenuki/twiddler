package tracker_test

import (
	"context"
	"testing"
	"time"

	"github.com/TeamTenuki/twiddler/clock"
	"github.com/TeamTenuki/twiddler/db"
	"github.com/TeamTenuki/twiddler/stream"
	"github.com/TeamTenuki/twiddler/testutil"
)

func TestStreamIsReported(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	baselineTime := clock.NowUTC()

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: baselineTime,
		},
	})

	tr.CloseAndWait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1")
}

func TestSameStreamTwiceInOneBatchReportedOnce(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	baselineTime := clock.NowUTC()

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

	tr.CloseAndWait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1")
}

func TestInterleavedStreamReportIsReportedOnlyOnce(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	baselineTime := clock.NowUTC()

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

	tr.CloseAndWait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1")
}

func TestStreamRestartIsNotReportedAfterEnd(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)

	fixedClock := clock.OverrideByFixed(time.Now())

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: clock.NowUTC(),
		},
	})

	tr.AwaitReport()
	fixedClock.Add(30 * time.Minute)

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream2",
			StartedAt: clock.NowUTC(),
		},
	})

	// Stream ends, due to eventual consistency we may get varied
	// reports on whether it's online or not.
	tr.Send([]stream.Stream{})
	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream2",
			StartedAt: clock.NowUTC(),
		},
	})

	tr.CloseAndWait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1")
}

func TestStreamRestartObservedWithinOneHourSingleReport(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	fixedClock := clock.OverrideByFixed(time.Now())

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: clock.NowUTC(),
		},
	})

	tr.AwaitReport()
	fixedClock.Add(30 * time.Minute)

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream2",
			StartedAt: clock.NowUTC(),
		},
	})

	tr.CloseAndWait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1")
}

func TestStreamWasObservedWithMoreThanHourGapBothReported(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)

	fixedClock := clock.OverrideByFixed(time.Now())

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: clock.NowUTC(),
		},
	})

	tr.AwaitReport()
	fixedClock.Add(time.Hour + time.Minute)

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream2",
			StartedAt: clock.NowUTC(),
		},
	})

	tr.CloseAndWait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1", "stream2")
}

func TestSameStreamOneHourLaterNotReported(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	fixedClock := clock.OverrideByFixed(time.Now())
	startedAt := clock.NowUTC()

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: startedAt,
		},
	})

	tr.Send([]stream.Stream{})
	tr.AwaitReport()
	fixedClock.Add(time.Hour + time.Minute)

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: startedAt,
		},
	})

	tr.CloseAndWait()

	testutil.LogReports(t, tr.C)

	store := tr.Room("room1")
	expectStreamReports(t, store.Streams, "stream1")
}

func TestObservedTimeIsUpdatedWhenSeeingStreamAgain(t *testing.T) {
	tr := testutil.NewTracker()
	setupDB(tr.C)
	fixedClock := clock.OverrideByFixed(time.Now())

	startedAt := clock.NowUTC()
	observed1 := clock.NowUTC()
	observed2 := observed1.Add(time.Hour)

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: startedAt,
		},
	})

	tr.AwaitReport()
	testutil.VerifyObservedAt(t, tr.C, observed1)
	fixedClock.Add(time.Hour)

	tr.Send([]stream.Stream{
		{
			User:      stream.User{ID: "user1"},
			ID:        "stream1",
			StartedAt: startedAt,
		},
	})

	tr.CloseAndWait()
	testutil.VerifyObservedAt(t, tr.C, observed2)

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
