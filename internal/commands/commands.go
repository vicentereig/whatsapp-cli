package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/vicente/whatsapp-cli/internal/client"
	"github.com/vicente/whatsapp-cli/internal/output"
	"github.com/vicente/whatsapp-cli/internal/store"
)

type App struct {
	client *client.WAClient
	store  *store.MessageStore
}

func NewApp(storeDir string) (*App, error) {
	cli, err := client.NewWAClient(storeDir)
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(storeDir, "messages.db")
	st, err := store.NewMessageStore(dbPath)
	if err != nil {
		return nil, err
	}

	return &App{
		client: cli,
		store:  st,
	}, nil
}

func (a *App) Close() {
	if a.client != nil {
		a.client.Disconnect()
	}
	if a.store != nil {
		a.store.Close()
	}
}

func (a *App) Auth(ctx context.Context) string {
	if a.client.IsAuthenticated() {
		return output.Success(map[string]interface{}{
			"authenticated": true,
			"message":       "Already authenticated",
		})
	}

	if err := a.client.Authenticate(ctx); err != nil {
		return output.Error(err)
	}

	return output.Success(map[string]interface{}{
		"authenticated": true,
		"message":       "Successfully authenticated",
	})
}

func (a *App) ListMessages(chatJID *string, query *string, limit, page int) string {
	messages, err := a.store.ListMessages(store.ListMessagesParams{
		ChatJID: chatJID,
		Query:   query,
		Limit:   limit,
		Page:    page,
	})
	if err != nil {
		return output.Error(err)
	}

	return output.Success(messages)
}

func (a *App) SearchContacts(query string) string {
	contacts, err := a.store.SearchContacts(query)
	if err != nil {
		return output.Error(err)
	}

	return output.Success(contacts)
}

func (a *App) ListChats(query *string, limit, page int) string {
	chats, err := a.store.ListChats(store.ListChatsParams{
		Query: query,
		Limit: limit,
		Page:  page,
	})
	if err != nil {
		return output.Error(err)
	}

	return output.Success(chats)
}

func (a *App) SendMessage(ctx context.Context, recipient, message string) string {
	if err := a.client.Connect(ctx); err != nil {
		return output.Error(err)
	}

	if err := a.client.SendMessage(ctx, recipient, message); err != nil {
		return output.Error(err)
	}

	// Store the message
	timestamp := time.Now()
	chatJID := recipient
	if !contains(recipient, "@") {
		chatJID = recipient + "@s.whatsapp.net"
	}

	// Store chat if needed
	a.store.StoreChat(chatJID, recipient, timestamp)
	a.store.StoreMessage(
		fmt.Sprintf("%d", timestamp.Unix()),
		chatJID,
		"me",
		message,
		timestamp,
		true,
		"", "", "", nil, nil, nil, 0,
	)

	return output.Success(map[string]interface{}{
		"sent":      true,
		"recipient": recipient,
		"message":   message,
	})
}

func contains(s, substr string) bool {
	for i := 0; i < len(s); i++ {
		if i+len(substr) <= len(s) && s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
