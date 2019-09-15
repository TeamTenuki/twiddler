package messenger

import (
	"context"

	"github.com/TeamTenuki/twiddler/stream"
)

type Messenger interface {
	MessageStream(c context.Context, roomID string, s *stream.Stream) error
	MessageStreamList(c context.Context, roomID string, s []stream.Stream) error
	MessageText(c context.Context, roomID string, t string) error
}
