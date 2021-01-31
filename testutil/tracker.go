package testutil

import (
	"context"
	"sync"

	"github.com/TeamTenuki/twiddler/db"
	"github.com/TeamTenuki/twiddler/stream"
	"github.com/TeamTenuki/twiddler/tracker"
)

type Tracker struct {
	C  context.Context
	w  *Watcher
	m  *Messenger
	wg *sync.WaitGroup
}

func NewTracker() *Tracker {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	w := NewWatcher()
	m := NewMessenger()
	c := setupDB()

	tracker := tracker.NewTracker(w, m)
	go func() {
		tracker.Track(c)
		wg.Done()
	}()

	return &Tracker{
		C:  c,
		w:  w,
		m:  m,
		wg: wg,
	}
}

func (t *Tracker) AwaitReport() {
	t.m.AwaitReport()
}

func (t *Tracker) Send(ss []stream.Stream) {
	t.w.Send(ss)
}

func (t *Tracker) CloseAndWait() {
	t.w.Close()
	t.wg.Wait()
}

func (t *Tracker) Room(roomID string) MessengerStore {
	return t.m.rooms[roomID]
}

func setupDB() context.Context {
	db.MustInit(":memory:")

	c := db.NewContext(context.Background())
	db.SetupDB(c)

	return c
}
