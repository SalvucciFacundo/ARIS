package telegram_test

import (
	"testing"

	"aris/internal/adapters/gateway/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestTelegramAuth_IsAuthorized(t *testing.T) {
	tests := []struct {
		name           string
		allowedChatIDs []int64
		allowedUserIDs []int64
		update         tgbotapi.Update
		expectedAuth   bool
	}{
		{
			name:           "Empty allowlists allows any user/chat",
			allowedChatIDs: []int64{},
			allowedUserIDs: []int64{},
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 12345},
					From: &tgbotapi.User{ID: 67890},
				},
			},
			expectedAuth: true,
		},
		{
			name:           "Allowed by Chat ID",
			allowedChatIDs: []int64{1001, 1002},
			allowedUserIDs: []int64{9001},
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 1001},
					From: &tgbotapi.User{ID: 5555},
				},
			},
			expectedAuth: true,
		},
		{
			name:           "Allowed by User ID",
			allowedChatIDs: []int64{1001},
			allowedUserIDs: []int64{9001, 9002},
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 5555},
					From: &tgbotapi.User{ID: 9002},
				},
			},
			expectedAuth: true,
		},
		{
			name:           "Rejected when neither Chat ID nor User ID in non-empty allowlists",
			allowedChatIDs: []int64{1001},
			allowedUserIDs: []int64{9001},
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: 5555},
					From: &tgbotapi.User{ID: 7777},
				},
			},
			expectedAuth: false,
		},
		{
			name:           "Nil message inside update is rejected",
			allowedChatIDs: []int64{},
			allowedUserIDs: []int64{},
			update:         tgbotapi.Update{Message: nil},
			expectedAuth:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorizer := telegram.NewAuthorizer(tt.allowedChatIDs, tt.allowedUserIDs)
			authorized := authorizer.IsAuthorized(&tt.update)
			if authorized != tt.expectedAuth {
				t.Errorf("expected authorized=%v, got %v", tt.expectedAuth, authorized)
			}
		})
	}
}
