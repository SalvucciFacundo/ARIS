package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Authorizer validates incoming Telegram updates against configured chat and user allowlists.
type Authorizer struct {
	allowedChats map[int64]struct{}
	allowedUsers map[int64]struct{}
	isRestricted bool
}

// NewAuthorizer creates a new Telegram authorizer.
func NewAuthorizer(chatIDs, userIDs []int64) *Authorizer {
	chats := make(map[int64]struct{}, len(chatIDs))
	for _, id := range chatIDs {
		chats[id] = struct{}{}
	}

	users := make(map[int64]struct{}, len(userIDs))
	for _, id := range userIDs {
		users[id] = struct{}{}
	}

	isRestricted := (len(chatIDs) > 0 || len(userIDs) > 0)

	return &Authorizer{
		allowedChats: chats,
		allowedUsers: users,
		isRestricted: isRestricted,
	}
}

// IsAuthorized returns true if the update is allowed to be processed.
func (a *Authorizer) IsAuthorized(update *tgbotapi.Update) bool {
	if update == nil || update.Message == nil {
		return false
	}

	// If no allowlists were configured, access is unrestricted.
	if !a.isRestricted {
		return true
	}

	chatID := update.Message.Chat.ID
	if _, ok := a.allowedChats[chatID]; ok {
		return true
	}

	if update.Message.From != nil {
		userID := update.Message.From.ID
		if _, ok := a.allowedUsers[userID]; ok {
			return true
		}
	}

	return false
}
