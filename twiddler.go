package twiddler

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/TeamTenuki/twiddler/config"
	"github.com/TeamTenuki/twiddler/db"
	"github.com/TeamTenuki/twiddler/tracker"
)

// Run is the main entry point that starts the bot interaction with the world.
// It manages cancellation through the c context parameter, i.e. Run will return
// when c.Done() is closed.
func Run(c context.Context, config *config.Config) error {
	db.SetupDB(c)

	dg, err := discordgo.New("Bot " + config.DiscordAPI)
	if err != nil {
		return fmt.Errorf("failed to create Discord client instance: %w", err)
	}

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		messageHandler(c, s, m)
	})

	if err := dg.Open(); err != nil {
		return fmt.Errorf("failed to open a WebSocket connection: %w", err)
	}

	r := newDiscordReporter(dg)
	p := newTwitchPoker(config.TwitchAPI, 2*time.Second)
	t := tracker.NewTracker(p, r)

	go t.Track(c)

	<-c.Done()

	return dg.Close()
}

func messageHandler(c context.Context, s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if mentionsBot(s, m.Mentions) {
		handleCommand(c, s, m)
	}
}

func mentionsBot(s *discordgo.Session, ms []*discordgo.User) bool {
	for _, u := range ms {
		if u.ID == s.State.User.ID {
			return true
		}
	}

	return false
}

type discordReporter struct {
	s *discordgo.Session
	r *strings.Replacer
}

func newDiscordReporter(s *discordgo.Session) tracker.Reporter {
	r := strings.NewReplacer("{width}", "1280", "{height}", "720")

	return &discordReporter{
		s: s,
		r: r,
	}
}

func (r *discordReporter) Report(c context.Context, roomID string, s *tracker.Stream) error {
	_, err := r.s.ChannelMessageSendEmbed(roomID, &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s Went Live!", s.UserName),
		Description: fmt.Sprintf("[%s](https://twitch.tv/%s)", s.Title, s.UserName),
		Image: &discordgo.MessageEmbedImage{
			URL:    r.r.Replace(s.Thumbnail),
			Width:  1280,
			Height: 720,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Live since %s", s.StartedAt),
		},
	})

	return err
}

func (r *discordReporter) ReportMessage(c context.Context, roomID string, content string) error {
	_, err := r.s.ChannelMessageSend(roomID, content)

	return err
}

// FIXME(destroycomputers): Factor out commands into separate file or package.
var commands = map[string]func(c context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) error{
	"list":   listCommand,
	"spam":   spamCommand,
	"forget": forgetCommand,
	"help":   helpCommand,
}

var commandRegex = regexp.MustCompile(`^<@\d+>\s+(\w+)(\s*[\w<>#]+)*$`)

func handleCommand(c context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error {
	groups := commandRegex.FindAllStringSubmatch(m.Content, -1)
	if groups == nil {
		return nil
	}

	command := strings.TrimSpace(groups[0][1])

	if handler, exists := commands[command]; exists {
		handler(c, s, m, groups[0][2:])
	}

	return nil
}

func listCommand(c context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	if true {
		s.ChannelMessageSend(m.ChannelID, "Nobody is currently streaming :pensive:")

		return nil
	}

	fields := make([]*discordgo.MessageEmbedField, 0)
	for _, stream := range make([]*tracker.Stream, 0) {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  stream.UserName,
			Value: fmt.Sprintf("[%s](https://twitch.tv/%s)", stream.Title, stream.UserName),
		})
	}

	s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
		Title:  "Currently Live",
		Fields: fields,
	})

	return nil
}

func spamCommand(c context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Command `spam` requires an argument - channel where it will spam")
		return nil
	}

	roomID, err := parseRoomID(args[0])
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, err.Error())
		return nil
	}

	db := db.FromContext(c)
	if _, err := db.ExecContext(c, `INSERT INTO [rooms] ([room_id]) VALUES (?)`, roomID); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			s.ChannelMessageSend(m.ChannelID,
				fmt.Sprintf("Failed to add channel <#%s>: it is already added.", roomID))
		}
		return nil
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Successfully added room <#%s>", roomID))

	return nil
}

func forgetCommand(c context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Command `forget` requires an argument - channel which to exclude from spamming")
		return nil
	}

	roomID, err := parseRoomID(args[0])
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, err.Error())
		return nil
	}

	db := db.FromContext(c)
	if _, err := db.ExecContext(c, `DELETE FROM [rooms] WHERE [room_id] = ?`, roomID); err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Failed to remove room <#%s> :pensive:", roomID))
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Successfully removed room <#%s>", roomID))

	return nil
}

func helpCommand(c context.Context, s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	_, err := s.ChannelMessageSend(m.ChannelID, "```\nUSAGE\n\tspam - Add channel to list of spammable channels\n\tforget - Remove channel from list of spammable channels\n\tlist - List currently live streamers\n\thelp - Display this message```")

	return err
}

func parseRoomID(s string) (string, error) {

	var roomRegex = regexp.MustCompile(`<#(\d+)>`)
	var groups = roomRegex.FindAllStringSubmatch(s, -1)
	if groups == nil {
		return "", errors.New("Improper room format")
	}

	var roomID = groups[0][1]

	return roomID, nil
}
