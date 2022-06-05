package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Room of a messenger that is waiting for reports on new streams.
type Room struct {
	// ID of a room in a messenger-specific format.
	ID string
}

// RoomsAll yields all the rooms from the DB.
func RoomsAll(c context.Context) ([]Room, error) {
	db := FromContext(c)

	ids := make([]string, 0)
	err := db.SelectContext(c, &ids, `SELECT [room_id] FROM [rooms]`)
	if err != nil {
		return nil, err
	}

	rooms := make([]Room, len(ids))
	for i := range ids {
		rooms[i] = Room{ID: ids[i]}
	}

	return rooms, err
}

// Report is a record of a successful report of a certain stream.
type Report struct {
	// Streamer ID.
	UserID string
	// ID of a particular stream.
	StreamID string
	// Timestamp of the stream start.
	StartedAt time.Time
	// Timestamp of the latest observation of the stream being live by twiddler.
	ObservedAt time.Time
}

// RawReport is a Report with unparsed timestamps. Used internally to retrieve rows from the DB.
//
// Not meant to be used by the client code.
type RawReport struct {
	StreamID   string `db:"stream_id"`
	UserID     string `db:"user_id"`
	StartedAt  string `db:"started_at"`
	ObservedAt string `db:"observed_at"`
}

// Cook converts a RawReport into a Report.
//
// Returns error if it fails to parse time.
func (r *RawReport) Cook() (Report, error) {
	startedAt, err := time.Parse(time.RFC3339, r.StartedAt)
	if err != nil {
		return Report{}, err
	}

	observedAt, err := time.Parse(time.RFC3339, r.ObservedAt)
	if err != nil {
		return Report{}, err
	}

	actual := Report{
		StreamID:   r.StreamID,
		UserID:     r.UserID,
		StartedAt:  startedAt,
		ObservedAt: observedAt,
	}

	return actual, nil
}

// ReportsAll yields all reports from the DB.
func ReportsAll(c context.Context) ([]Report, error) {
	rawReports := make([]RawReport, 0)
	err := db.SelectContext(
		c,
		&rawReports,
		`SELECT [stream_id], [user_id], [started_at], [observed_at] FROM [reports]`,
	)
	if err != nil {
		return nil, err
	}

	reports := make([]Report, len(rawReports))
	for i := range rawReports {
		cooked, err := rawReports[i].Cook()
		if err != nil {
			return nil, err
		}
		reports[i] = cooked
	}

	return reports, nil
}

// ReportFor select a report for the given streamID and startedAt.
//
// If there is no such report, sql.ErrNoRows is propagated as a return value.
func ReportFor(c context.Context, streamID string, startedAt time.Time) (Report, error) {
	db := FromContext(c)

	var raw RawReport
	err := db.GetContext(
		c,
		&raw,
		`SELECT [stream_id], [user_id], [started_at], [observed_at] FROM [reports] WHERE [stream_id] = ? AND [started_at] = ?`,
		streamID,
		startedAt.Format(time.RFC3339),
	)
	if err != nil {
		return Report{}, err
	}

	return raw.Cook()
}

// ReportStore stores a Report about a successful stream going live report.
func ReportStore(c context.Context, r Report) error {
	db := FromContext(c)

	_, err := db.ExecContext(
		c,
		`INSERT INTO [reports] ([user_id], [stream_id], [started_at], [observed_at]) VALUES (?, ?, ?, ?)`,
		r.UserID,
		r.StreamID,
		r.StartedAt.Format(time.RFC3339),
		r.ObservedAt.Format(time.RFC3339),
	)

	return err
}

// ReportObserveForStreams will update [observed_at] for every given stream.
func ReportObserveForStreams(c context.Context, streamIDs []string, at time.Time) error {
	db := FromContext(c)

	for i := range streamIDs {
		streamIDs[i] = "'" + streamIDs[i] + "'"
	}

	_, err := db.ExecContext(
		c,
		fmt.Sprintf(
			`UPDATE [reports] SET [observed_at] = ? WHERE [stream_id] IN (%s)`,
			strings.Join(streamIDs, ", "),
		),
		at.Format(time.RFC3339),
	)

	return err
}

// ReportWasReported answers whether a certain stream was ever successfully reported.
func ReportWasReported(c context.Context, streamID string) (bool, error) {
	db := FromContext(c)

	err := db.GetContext(
		c,
		new(string),
		`SELECT [started_at] FROM [reports] WHERE [stream_id] = ? LIMIT 1`,
		streamID,
	)

	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// ReportLatestByUser yields a latest report for a particular user.
//
// If there is no any reports, sql.ErrNoRows is propagated as a return value.
func ReportLatestByUser(c context.Context, userID string) (Report, error) {
	db := FromContext(c)

	var raw RawReport
	err := db.GetContext(
		c,
		&raw,
		`SELECT
			[user_id]
			, [stream_id]
			, [started_at]
			, [observed_at]
		FROM
			[reports]
		WHERE
			[user_id] = ?
		ORDER BY datetime([observed_at]) DESC
		LIMIT 1`,
		userID,
	)

	if err != nil {
		return Report{}, err
	}

	return raw.Cook()
}
