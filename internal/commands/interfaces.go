// Package commands provides the CLI command implementations.
//
// # Dependency Injection
//
// The interfaces below define the dependencies of App, enabling testability
// through mock injection. Types are shared via internal/types to avoid
// circular dependencies.
//
// Usage:
//   - Production: Use NewApp() which creates concrete implementations
//   - Testing: Use NewAppWithDeps() to inject mocks
package commands

import (
	"context"
	"time"

	"github.com/vicentereig/whatsapp-cli/internal/store"
	"github.com/vicentereig/whatsapp-cli/internal/types"
)

// MessageStore defines the interface for message persistence.
// The concrete implementation is store.MessageStore.
// Defined here (at consumer) per Go best practice: "Accept interfaces, return concrete types"
type MessageStore interface {
	ListMessages(params store.ListMessagesParams) ([]store.Message, error)
	SearchContacts(query string) ([]store.Contact, error)
	ListChats(params store.ListChatsParams) ([]store.Chat, error)
	StoreChat(jid, name string, lastMessageTime time.Time) error
	StoreMessage(id, chatJID, sender, content string, timestamp time.Time, isFromMe bool,
		mediaType, filename, url, directPath, mimeType string,
		mediaKey, fileSHA256, fileEncSHA256 []byte, fileLength uint64) error
	GetMessageForDownload(id string, chatJID *string) (store.MessageDownloadInfo, error)
	MarkMediaDownloaded(id, chatJID, localPath string, downloadedAt time.Time) error
	Close() error
}

// WAClient defines the interface for WhatsApp client operations.
// The concrete implementation is client.WAClient.
type WAClient interface {
	IsAuthenticated() bool
	Authenticate(ctx context.Context) error
	Connect(ctx context.Context) error
	Disconnect()
	SendMessage(ctx context.Context, recipient, message string) error
	ResolveChatName(ctx context.Context, jid string, evt interface{}) string
	DownloadMediaToFile(ctx context.Context, req types.MediaDownloadRequest, targetPath string) (int64, error)
	StartSync(ctx context.Context, eventHandler func(interface{})) error
}
