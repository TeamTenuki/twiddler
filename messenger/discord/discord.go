package discord

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/TeamTenuki/twiddler/stream"
)

type Messenger struct {
	s *discordgo.Session
}

func (m *Messenger) MessageStream(c context.Context, roomID string, s *stream.Stream) error {
	_, err := m.s.ChannelMessageSendEmbed(roomID, &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s Went Live!", s.User.Name),
		// FIXME(destroycomputers): Replace this link with s.ChannelURL or something.
		Description: fmt.Sprintf("[%s](https://twitch.tv/%s)", s.Title, s.User.Name),
		Image: &discordgo.MessageEmbedImage{
			URL:    s.ThumbnailURL.String(),
			Width:  1280,
			Height: 720,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Live since %s", s.StartedAt.Format(time.RFC3339)),
		},
	})

	return err
}
