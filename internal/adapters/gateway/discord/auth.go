package discord

import (
	"github.com/bwmarrin/discordgo"
)

// Authorizer checks incoming Discord message events against channel/user allowlists.
type Authorizer struct {
	allowedChannels map[string]struct{}
	allowedUsers    map[string]struct{}
	isRestricted    bool
}

// NewAuthorizer creates a new Discord access authorizer.
func NewAuthorizer(channelIDs, userIDs []string) *Authorizer {
	channels := make(map[string]struct{}, len(channelIDs))
	for _, id := range channelIDs {
		channels[id] = struct{}{}
	}

	users := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		users[id] = struct{}{}
	}

	isRestricted := (len(channelIDs) > 0 || len(userIDs) > 0)

	return &Authorizer{
		allowedChannels: channels,
		allowedUsers:    users,
		isRestricted:    isRestricted,
	}
}

// IsAuthorized evaluates whether a Discord message should be processed.
func (a *Authorizer) IsAuthorized(m *discordgo.MessageCreate) bool {
	if m == nil || m.Message == nil || m.Author == nil {
		return false
	}

	// Never process messages originating from bots (including self)
	if m.Author.Bot {
		return false
	}

	// If no allowlists were configured, access is unrestricted for human users
	if !a.isRestricted {
		return true
	}

	if _, ok := a.allowedChannels[m.ChannelID]; ok {
		return true
	}

	if _, ok := a.allowedUsers[m.Author.ID]; ok {
		return true
	}

	return false
}
