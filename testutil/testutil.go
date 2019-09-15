package testutil

import (
	"context"
	"testing"

	"github.com/TeamTenuki/twiddler/db"
)

type report struct {
	StreamID  string `db:"stream_id"`
	UserID    string `db:"user_id"`
	StartedAt string `db:"started_at"`
}

func LogReports(t *testing.T, c context.Context) {
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
