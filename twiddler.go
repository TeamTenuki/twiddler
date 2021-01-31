package twiddler

import (
	"context"
	"time"

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

	m, err := discord.NewMessenger(config.DiscordAPI)
	if err != nil {
		return err
	}

	f := twitch.NewFetcher(c, config.TwitchClientID, config.TwitchSecret)
	w := watcher.Periodic(f, 8*time.Second)
	t := tracker.NewTracker(w, m)

	m.AddCommandHandler(c, commands.NewHandler(t))
	if err := m.Run(); err != nil {
		return err
	}

	t.Track(c)

	return m.Close()
}
