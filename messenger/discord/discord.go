package discord

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/TeamTenuki/twiddler/messenger"
	"github.com/TeamTenuki/twiddler/stream"
)

type Messenger struct {
	s *discordgo.Session
}

func NewMessenger(s *discordgo.Session) messenger.Messenger {
	return &Messenger{
		s: s,
	}
}

func (m *Messenger) MessageStream(c context.Context, roomID string, s *stream.Stream) error {
	title := fmt.Sprintf("%s Went Live!", s.User.DisplayName)
	if strings.ToLower(s.User.Name) != strings.ToLower(s.User.DisplayName) {
		title = fmt.Sprintf("%s (%s) Went Live!",
			strings.Replace(s.User.DisplayName, "_", "\\_", -1),
			strings.Replace(s.User.Name, "_", "\\_", -1))
	}

	thumbnailURL := fmt.Sprintf("%s?cache_invalidation_token=%d", s.ThumbnailURL, rand.Int())

	_, err := m.s.ChannelMessageSendEmbed(roomID, &discordgo.MessageEmbed{
		Title:       title,
		Description: fmt.Sprintf("[%s](%s)", s.Title, s.User.ChannelURL),
		Image: &discordgo.MessageEmbedImage{
			URL:    thumbnailURL,
			Width:  1280,
			Height: 720,
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL:    s.User.PictureURL.String(),
			Width:  300,
			Height: 300,
		},
		Color: 0x00aa00,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "Twitch",
			URL:     s.User.ChannelURL.String(),
			IconURL: "https://assets.help.twitch.tv/Glitch_Purple_RGB.png",
		},
		Timestamp: s.StartedAt.Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Live since",
		},
	})

	return err
}

func (m *Messenger) MessageStreamList(c context.Context, roomID string, s []stream.Stream) error {
	fields := make([]*discordgo.MessageEmbedField, 0)
	for _, stream := range s {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  stream.User.Name,
			Value: fmt.Sprintf("[%s](%s)", stream.Title, stream.User.ChannelURL),
		})
	}

	_, err := m.s.ChannelMessageSendEmbed(roomID, &discordgo.MessageEmbed{
		Title:  "Currently Live",
		Fields: fields,
	})

	return err
}

func (m *Messenger) MessageText(c context.Context, roomID, text string) error {
	_, err := m.s.ChannelMessageSend(roomID, text)

	return err
}
