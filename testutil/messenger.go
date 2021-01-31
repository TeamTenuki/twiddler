package testutil

import (
	"context"

	"github.com/TeamTenuki/twiddler/messenger"
	"github.com/TeamTenuki/twiddler/stream"
)

type MessengerStore struct {
	Streams  []*stream.Stream
	Messages []string
}

var _ messenger.Messenger = &Messenger{}

type Messenger struct {
	rooms   map[string]MessengerStore
	awaiter chan struct{}
}

func NewMessenger() *Messenger {
	return &Messenger{
		rooms:   make(map[string]MessengerStore),
		awaiter: make(chan struct{}, 1000),
	}
}

func (r *Messenger) MessageStream(c context.Context, roomID string, s *stream.Stream) error {
	store := r.rooms[roomID]
	store.Streams = append(store.Streams, s)

	r.rooms[roomID] = store
	r.awaiter <- struct{}{}

	return nil
}

func (r *Messenger) MessageStreamList(c context.Context, roomID string, ss []stream.Stream) error {
	r.awaiter <- struct{}{}
	return nil
}

func (r *Messenger) MessageText(c context.Context, roomID string, content string) error {
	store := r.rooms[roomID]
	store.Messages = append(store.Messages, content)

	r.rooms[roomID] = store
	r.awaiter <- struct{}{}

	return nil
}

func (r *Messenger) AddCommandHandler(c context.Context, h messenger.Handler) {

}

func (r *Messenger) Run() error {
	return nil
}

func (r *Messenger) Close() error {
	return nil
}

func (r *Messenger) AwaitReport() {
	<-r.awaiter
}
