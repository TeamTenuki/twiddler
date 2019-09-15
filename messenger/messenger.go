package messenger

import (
	"context"

	"github.com/TeamTenuki/twiddler/stream"
)

// Messenger is a generic messenger that knows how to format a message containing
// information about a stream, stream list or send an arbitrary text message.
type Messenger interface {
	// MessageStream knows how to format and send a message
	// containing information about a single stream.
	MessageStream(c context.Context, roomID string, s *stream.Stream) error

	// MessageStreamList knows how to format and send a message
	// containing information about a list of streams.
	MessageStreamList(c context.Context, roomID string, s []stream.Stream) error

	// MessageText knows how to send an arbitrary text message.
	MessageText(c context.Context, roomID string, t string) error
}
