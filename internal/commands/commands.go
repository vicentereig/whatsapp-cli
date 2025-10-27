package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vicente/whatsapp-cli/internal/client"
	"github.com/vicente/whatsapp-cli/internal/output"
	"github.com/vicente/whatsapp-cli/internal/store"
	"go.mau.fi/whatsmeow/types/events"
)

type App struct {
	client  *client.WAClient
	store   *store.MessageStore
	version string
}

func NewApp(storeDir, version string) (*App, error) {
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
		client:  cli,
		store:   st,
		version: resolveVersion(version, gitDescribe),
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

	// Resolve a friendly chat name when available (falls back to JID/recipient)
	chatName := a.client.ResolveChatName(ctx, chatJID, nil)
	if chatName == "" {
		chatName = recipient
	}

	// Store chat if needed
	a.store.StoreChat(chatJID, chatName, timestamp)
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

// Sync connects to WhatsApp and continuously syncs messages to the database
func (a *App) Sync(ctx context.Context) string {
	messageCount := 0

	version := a.version
	if strings.TrimSpace(version) == "" {
		version = "unknown"
	}
	fmt.Fprintf(os.Stderr, "ℹ️  whatsapp-cli version: %s\n", version)

	// Create event handler
	eventHandler := func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			// Extract message details
			id, chatJID, sender, content, timestamp, isFromMe, mediaType, filename, url := client.HandleMessage(v)

			chatName := a.client.ResolveChatName(ctx, chatJID, v)
			if chatName == "" && chatJID != "" {
				chatName = chatJID
			}

			// Store chat
			msgTime := time.Unix(timestamp, 0)
			a.store.StoreChat(chatJID, chatName, msgTime)

			// Store message
			a.store.StoreMessage(
				id,
				chatJID,
				sender,
				content,
				msgTime,
				isFromMe,
				mediaType,
				filename,
				url,
				nil, nil, nil, 0,
			)

			messageCount++
			fmt.Fprintf(os.Stderr, "\r💬 Synced %d messages...", messageCount)

		case *events.HistorySync:
			fmt.Fprintf(os.Stderr, "\n📜 Processing history sync (%d conversations)...\n", len(v.Data.Conversations))
			for _, conv := range v.Data.Conversations {
				chatJID := conv.GetId()
				chatName := conv.GetName()
				if chatName == "" {
					chatName = a.client.ResolveChatName(ctx, chatJID, nil)
					if chatName == "" {
						chatName = chatJID
					}
				}

				// Process messages in this conversation
				for _, msg := range conv.Messages {
					if msg.Message == nil {
						continue
					}

					histMsg := msg.Message
					msgID := histMsg.Key.GetId()
					sender := histMsg.Key.GetParticipant()
					if sender == "" {
						sender = histMsg.Key.GetRemoteJid()
					}
					isFromMe := histMsg.Key.GetFromMe()
					timestamp := time.Unix(int64(histMsg.GetMessageTimestamp()), 0)

					// Extract content
					content := ""
					mediaType := ""
					filename := ""
					url := ""

					if histMsg.Message.GetConversation() != "" {
						content = histMsg.Message.GetConversation()
					} else if extText := histMsg.Message.GetExtendedTextMessage(); extText != nil {
						content = extText.GetText()
					} else if img := histMsg.Message.GetImageMessage(); img != nil {
						mediaType = "image"
						content = img.GetCaption()
						filename = img.GetCaption()
						url = img.GetURL()
					} else if video := histMsg.Message.GetVideoMessage(); video != nil {
						mediaType = "video"
						content = video.GetCaption()
						filename = video.GetCaption()
						url = video.GetURL()
					} else if audio := histMsg.Message.GetAudioMessage(); audio != nil {
						mediaType = "audio"
						content = "[Audio]"
						url = audio.GetURL()
					} else if doc := histMsg.Message.GetDocumentMessage(); doc != nil {
						mediaType = "document"
						content = doc.GetCaption()
						filename = doc.GetFileName()
						url = doc.GetURL()
					}

					// Store chat
					a.store.StoreChat(chatJID, chatName, timestamp)

					// Store message
					a.store.StoreMessage(
						msgID,
						chatJID,
						sender,
						content,
						timestamp,
						isFromMe,
						mediaType,
						filename,
						url,
						nil, nil, nil, 0,
					)

					messageCount++
				}
			}
			fmt.Fprintf(os.Stderr, "\r💬 Synced %d messages...", messageCount)

		case *events.Connected:
			fmt.Fprintln(os.Stderr, "\n✓ Connected to WhatsApp")
			fmt.Fprintln(os.Stderr, "🔄 Listening for messages... (Press Ctrl+C to stop)")

		case *events.Disconnected:
			fmt.Fprintln(os.Stderr, "\n⚠ Disconnected from WhatsApp")
		}
	}

	// Start syncing
	fmt.Fprintln(os.Stderr, "🚀 Starting WhatsApp sync...")
	if err := a.client.StartSync(ctx, eventHandler); err != nil {
		return output.Error(err)
	}

	// Wait for context cancellation (Ctrl+C)
	<-ctx.Done()

	fmt.Fprintf(os.Stderr, "\n\n✓ Sync completed. Total messages synced: %d\n", messageCount)

	return output.Success(map[string]interface{}{
		"synced":         true,
		"messages_count": messageCount,
	})
}

func resolveVersion(version string, describeFn func() (string, error)) string {
	if strings.TrimSpace(version) != "" && version != "dev" {
		return version
	}

	if describeFn != nil {
		if gitVersion, err := describeFn(); err == nil && strings.TrimSpace(gitVersion) != "" {
			return gitVersion
		}
	}

	if strings.TrimSpace(version) == "" {
		return "unknown"
	}
	return version
}

func gitDescribe() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--dirty", "--always")
	cmd.Env = os.Environ()
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
