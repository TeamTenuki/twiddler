package twiddler

import (
	"context"

	"github.com/TeamTenuki/twiddler/db"
)

func setupDB(c context.Context) {
	db := db.FromContext(c)

	db.MustExecContext(c, `CREATE TABLE IF NOT EXISTS [rooms] ([room_id] TEXT NOT NULL, UNIQUE ([room_id]))`)
	db.MustExecContext(c, `CREATE TABLE IF NOT EXISTS [reports] (
		[channel_id] TEXT NOT NULL,
		[started_at] TEXT NOT NULL,
		UNIQUE ([channel_id], [started_at])
	)`)
}
