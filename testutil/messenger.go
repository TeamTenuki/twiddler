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
	rooms map[string]MessengerStore
}

func NewMessenger() *Messenger {
	return &Messenger{
		rooms: make(map[string]MessengerStore),
	}
}

func (r *Messenger) MessageStream(c context.Context, roomID string, s *stream.Stream) error {
	store := r.rooms[roomID]
	store.Streams = append(store.Streams, s)

	r.rooms[roomID] = store

	return nil
}

func (r *Messenger) MessageStreamList(c context.Context, roomID string, ss []stream.Stream) error {
	return nil
}

func (r *Messenger) MessageText(c context.Context, roomID string, content string) error {
	store := r.rooms[roomID]
	store.Messages = append(store.Messages, content)

	r.rooms[roomID] = store

	return nil
}
