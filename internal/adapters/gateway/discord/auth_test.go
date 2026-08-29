package discord_test

import (
	"testing"

	"aris/internal/adapters/gateway/discord"
	"github.com/bwmarrin/discordgo"
)

func TestDiscordAuth_IsAuthorized(t *testing.T) {
	tests := []struct {
		name              string
		allowedChannelIDs []string
		allowedUserIDs    []string
		msg               *discordgo.MessageCreate
		expectedAuth      bool
	}{
		{
			name:              "Bot messages are always rejected",
			allowedChannelIDs: []string{},
			allowedUserIDs:    []string{},
			msg: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ChannelID: "111",
					Author: &discordgo.User{
						ID:  "222",
						Bot: true,
					},
				},
			},
			expectedAuth: false,
		},
		{
			name:              "Empty allowlists allow any human user",
			allowedChannelIDs: []string{},
			allowedUserIDs:    []string{},
			msg: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ChannelID: "111",
					Author: &discordgo.User{
						ID:  "222",
						Bot: false,
					},
				},
			},
			expectedAuth: true,
		},
		{
			name:              "Allowed by Channel ID",
			allowedChannelIDs: []string{"chan-1", "chan-2"},
			allowedUserIDs:    []string{"user-9"},
			msg: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ChannelID: "chan-1",
					Author: &discordgo.User{
						ID:  "random-user",
						Bot: false,
					},
				},
			},
			expectedAuth: true,
		},
		{
			name:              "Allowed by User ID in unlisted channel",
			allowedChannelIDs: []string{"chan-1"},
			allowedUserIDs:    []string{"user-9"},
			msg: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ChannelID: "chan-other",
					Author: &discordgo.User{
						ID:  "user-9",
						Bot: false,
					},
				},
			},
			expectedAuth: true,
		},
		{
			name:              "Rejected when neither Channel ID nor User ID is in non-empty allowlists",
			allowedChannelIDs: []string{"chan-1"},
			allowedUserIDs:    []string{"user-9"},
			msg: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ChannelID: "chan-other",
					Author: &discordgo.User{
						ID:  "user-stranger",
						Bot: false,
					},
				},
			},
			expectedAuth: false,
		},
		{
			name:              "Nil message rejected",
			allowedChannelIDs: []string{},
			allowedUserIDs:    []string{},
			msg:               nil,
			expectedAuth:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := discord.NewAuthorizer(tt.allowedChannelIDs, tt.allowedUserIDs)
			authorized := authorizer.IsAuthorized(tt.msg)
			if authorized != tt.expectedAuth {
				t.Errorf("expected authorized=%v, got %v", tt.expectedAuth, authorized)
			}
		})
	}
}
