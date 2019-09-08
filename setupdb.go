package twiddler

import (
	"context"

	"github.com/TeamTenuki/twiddler/db"
)

func setupDB(c context.Context) {
	db := db.FromContext(c)

	db.MustExecContext(c, `CREATE TABLE IF NOT EXISTS [rooms] (
		[room_id] TEXT NOT NULL, UNIQUE ([room_id])
	)`)

	db.MustExecContext(c, `CREATE TABLE IF NOT EXISTS [reports] (
		[user_id]    TEXT NOT NULL,
		[stream_id]  TEXT NOT NULL,
		[started_at] TEXT NOT NULL,

		UNIQUE ([stream_id], [started_at])
	)`)
}
