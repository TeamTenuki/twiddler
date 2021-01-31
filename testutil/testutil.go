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

func VerifyObservedAt(t *testing.T, c context.Context, expectedTime time.Time) {
	t.Helper()

	db := db.FromContext(c)

	var rep report
	err := db.GetContext(c, &rep, `SELECT [stream_id], [user_id], [started_at], [observed_at] FROM [reports]`)
	if err != nil {
		t.Errorf("Failed to retrieve reports: %s", err)
	}

	dt, err := time.Parse(time.RFC3339, rep.ObservedAt)
	if err != nil {
		t.Errorf("Failed to parse observed_at: %s", err)
	}

	if dt != expectedTime.Truncate(time.Second) {
		t.Errorf("Time mismatch, expected %q, got %q", expectedTime, dt)
	}
}

func LogReports(t *testing.T, c context.Context) {
	t.Helper()

	db := db.FromContext(c)

	var reports []report
	err := db.Select(&reports, `SELECT [stream_id], [user_id], [started_at], [observed_at] FROM [reports]`)
	if err != nil {
		t.Errorf("Failed to retrieve reports: %s", err)
	}

	for i := range reports {
		t.Logf("%#v", reports[i])
	}
}
