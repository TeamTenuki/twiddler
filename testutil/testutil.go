package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/TeamTenuki/twiddler/db"
)

type report struct {
	StreamID   string `db:"stream_id"`
	UserID     string `db:"user_id"`
	StartedAt  string `db:"started_at"`
	ObservedAt string `db:"observed_at"`
}

func VerifyObservedAt(t *testing.T, c context.Context, streamID string, startedAt time.Time, expectedTime time.Time) {
	t.Helper()

	rep, err := db.ReportFor(c, streamID, startedAt)
	if err != nil {
		t.Errorf("Failed to retrieve reports: %s", err)
		return
	}

	if rep.ObservedAt != expectedTime.Truncate(time.Second) {
		t.Errorf("Time mismatch, expected %q, got %q", expectedTime, rep.ObservedAt)
	}
}

func LogReports(t *testing.T, c context.Context) {
	t.Helper()

	reps, err := db.ReportsAll(c)
	if err != nil {
		t.Errorf("Failed to retrieve reports: %s", err)
		return
	}

	for _, rep := range reps {
		t.Logf("%#v", rep)
	}
}
