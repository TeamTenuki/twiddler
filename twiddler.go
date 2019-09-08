package twiddler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/TeamTenuki/twiddler/config"
	"github.com/TeamTenuki/twiddler/db"
)

// Run is the main entry point that starts the bot interaction with the world.
// It manages cancellation through the c context parameter, i.e. Run will return
// when c.Done() is closed.
func Run(c context.Context, config *config.Config) error {
	setupDB(c)

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

	streamC := streamSupply(config.TwitchAPI)

	go streamHandler(c, dg, streamC)

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

// Stream describes relevant information about a Twitch channel.
type Stream struct {
	// Twitch username of the channel owner.
	UserName string `json:"user_name"`

	// Twitch user ID.
	UserID string `json:"user_id"`

	// Channel title.
	Title string `json:"title"`

	// Live stream thumbnail URL.
	Thumbnail string `json:"thumbnail_url"`

	// Unique stream identifier.
	ID string `json:"id" db:"stream_id"`

	// ISO-8601 date/time of stream going live.
	StartedAt string `json:"started_at" db:"started_at"`
}

// StreamContainer represents unmarshalled JSON response from Twitch.
type StreamContainer struct {
	Data       []Stream `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor,omitempty"`
	} `json:"pagination,omitempty"`
}

var currentlyLive []Stream

// FIXME(destroycomputers): Refactor this garbage bin.
func streamHandler(c context.Context, s *discordgo.Session, streamC chan []Stream) {
	db := db.FromContext(c)

	// TODO(destroycomputers): Consider making this configurable.
	rep := strings.NewReplacer("{width}", "1280", "{height}", "720")

	for streams := range streamC {
		select {
		default:
		case <-c.Done():
			close(streamC)
			continue
		}

		reportableStreams := make([]Stream, 0)

	outer:
		for _, stream := range streams {
			for _, liveStream := range currentlyLive {
				if stream.ID == liveStream.ID {
					continue outer
				}
			}

			// Default date/time that will be used if there are no rows for given channel.
			var alreadyReported = "2006-01-02T15:04:05Z"

			err := db.GetContext(c, &alreadyReported, `SELECT [started_at]
			FROM [reports]
			WHERE [user_id] = ?
			ORDER BY datetime([started_at]) DESC
			LIMIT 1`,
				stream.UserID)

			if err != nil && err != sql.ErrNoRows {
				log.Printf("Failed to retrieve rows from DB: %s", err)
				return // This error may indicate a broken connection, we need to restart program.
			}

			t, err := time.Parse(time.RFC3339, alreadyReported)
			if err != nil {
				log.Printf("Failed to parse date/time from reported stream: %s", err)
				continue
			}

			if time.Since(t) > time.Hour {
				reportableStreams = append(reportableStreams, stream)
			}
		}

		var spammableRooms []string
		db.SelectContext(c, &spammableRooms, `SELECT [room_id] FROM [rooms]`)

		for _, room := range spammableRooms {
			for _, stream := range reportableStreams {
				channel, err := s.Channel(room)
				if err != nil {
					log.Printf("Failed to retrieve channel %q: %s", room, err)
				}

				s.ChannelMessageSendEmbed(channel.ID, &discordgo.MessageEmbed{
					Title:       fmt.Sprintf("%s Went Live!", stream.UserName),
					Description: fmt.Sprintf("[%s](https://twitch.tv/%s)", stream.Title, stream.UserName),
					Image: &discordgo.MessageEmbedImage{
						URL:    rep.Replace(stream.Thumbnail),
						Width:  1280,
						Height: 720,
					},
					Footer: &discordgo.MessageEmbedFooter{
						Text: fmt.Sprintf("Live since %s", stream.StartedAt),
					},
				})
			}
		}

		for _, stream := range reportableStreams {
			_, err := db.ExecContext(c, `INSERT INTO [reports]
			([user_id], [stream_id], [started_at]) VALUES (?, ?, ?)`,
				stream.UserID,
				stream.ID,
				stream.StartedAt)
			if err != nil {
				log.Printf("Failed to add row for %q %q %q: %s",
					stream.UserID, stream.ID, stream.StartedAt, err)
			}
		}

		currentlyLive = streams
	}
}

func streamSupply(twitchAPI string) chan []Stream {
	c := make(chan []Stream)

	go func() {
		for tick := range time.NewTicker(2 * time.Second).C {
			req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/streams?game_id=65360&first=100", nil)
			if err != nil {
				log.Panicf("[%s] Unexpected error: %s", tick, err)
			}
			req.Header.Add("Client-ID", twitchAPI)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("[%s] Failed to perform HTTP request: %s", tick, err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				var streamContainer StreamContainer

				if err := json.NewDecoder(resp.Body).Decode(&streamContainer); err != nil {
					log.Printf("[%s] Failed to decode JSON: %s", tick, err)
					continue
				}

				c <- streamContainer.Data
			}
		}
	}()

	return c
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
	if len(currentlyLive) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Nobody is currently streaming :pensive:")

		return nil
	}

	fields := make([]*discordgo.MessageEmbedField, 0)
	for _, stream := range currentlyLive {
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
