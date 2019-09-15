package twiddler

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/TeamTenuki/twiddler/commands"
	"github.com/TeamTenuki/twiddler/config"
	"github.com/TeamTenuki/twiddler/db"
	"github.com/TeamTenuki/twiddler/messenger/discord"
	"github.com/TeamTenuki/twiddler/stream/twitch"
	"github.com/TeamTenuki/twiddler/tracker"
	"github.com/TeamTenuki/twiddler/watcher"
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

	m := discord.NewMessenger(dg)
	f := twitch.NewFetcher(config.TwitchAPI)
	w := watcher.Periodic(f, 2*time.Second)
	t := tracker.NewTracker(w, m)
	h := commands.NewHandler(t)

	dg.AddHandler(func(s *discordgo.Session, mc *discordgo.MessageCreate) {
		if mc.Author.ID == s.State.User.ID {
			return
		}

		if mentionsBot(s, mc.Mentions) {
			h.Handle(c, mc.ChannelID, mc.Content, m)
		}
	})

	if err := dg.Open(); err != nil {
		return fmt.Errorf("failed to open a WebSocket connection: %w", err)
	}

	t.Track(c)

	return dg.Close()
}

func mentionsBot(s *discordgo.Session, ms []*discordgo.User) bool {
	for _, u := range ms {
		if u.ID == s.State.User.ID {
			return true
		}
	}

	return false
}
