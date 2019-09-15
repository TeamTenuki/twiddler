package commands

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/TeamTenuki/twiddler/db"
	"github.com/TeamTenuki/twiddler/messenger"
	"github.com/TeamTenuki/twiddler/stream"
)

type Command = func(c context.Context, sourceID string, args []string, m messenger.Messenger) error

type StreamingState interface {
	Live() []stream.Stream
}

type Handler struct {
	commands map[string]Command
	state    StreamingState
}

func NewHandler(state StreamingState) *Handler {
	h := &Handler{state: state}

	h.commands = map[string]Command{
		"list":   h.listCommand,
		"spam":   h.spamCommand,
		"forget": h.forgetCommand,
		"help":   h.helpCommand,
	}

	return h
}

var commandRegex = regexp.MustCompile(`^<@\d+>\s+(\w+)(\s*[\w<>#]+)*$`)

func (h *Handler) Handle(c context.Context, sourceID, message string, m messenger.Messenger) error {
	groups := commandRegex.FindAllStringSubmatch(message, -1)
	if groups == nil {
		return nil
	}

	command := strings.TrimSpace(groups[0][1])
	args := groups[0][2:]

	if handler, exists := h.commands[command]; exists {
		return handler(c, sourceID, args, m)
	}

	return nil
}

func (h *Handler) listCommand(c context.Context, sourceID string, args []string, m messenger.Messenger) error {
	streams := h.state.Live()

	if len(streams) == 0 {
		return m.MessageText(c, sourceID, "Nobody is currently streaming :pensive:")
	}

	return m.MessageStreamList(c, sourceID, streams)
}

func (h *Handler) spamCommand(c context.Context, sourceID string, args []string, m messenger.Messenger) error {
	if len(args) == 0 {
		return m.MessageText(c, sourceID, "Command `spam` requires an argument - channel where it will spam")
	}

	roomID, err := parseRoomID(args[0])
	if err != nil {
		return m.MessageText(c, sourceID, err.Error())
	}

	db := db.FromContext(c)
	if _, err := db.ExecContext(c, `INSERT INTO [rooms] ([room_id]) VALUES (?)`, roomID); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return m.MessageText(c, sourceID,
				fmt.Sprintf("Failed to add channel <#%s>: it is already added.", roomID))
		}
		return nil
	}

	m.MessageText(c, sourceID, fmt.Sprintf("Successfully added room <#%s>", roomID))

	return nil
}

func (h *Handler) forgetCommand(c context.Context, sourceID string, args []string, m messenger.Messenger) error {
	if len(args) == 0 {
		return m.MessageText(c, sourceID, "Command `forget` requires an argument - channel which to exclude from spamming")
	}

	roomID, err := parseRoomID(args[0])
	if err != nil {
		return m.MessageText(c, sourceID, err.Error())
	}

	db := db.FromContext(c)
	if _, err := db.ExecContext(c, `DELETE FROM [rooms] WHERE [room_id] = ?`, roomID); err != nil {
		m.MessageText(c, sourceID, fmt.Sprintf("Failed to remove room <#%s> :pensive:", roomID))
	}

	return m.MessageText(c, sourceID, fmt.Sprintf("Successfully removed room <#%s>", roomID))
}

func (h *Handler) helpCommand(c context.Context, sourceID string, args []string, m messenger.Messenger) error {
	return m.MessageText(c, sourceID, "```\nUSAGE\n\tspam - Add channel to list of spammable channels\n\tforget - Remove channel from list of spammable channels\n\tlist - List currently live streamers\n\thelp - Display this message```")
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
