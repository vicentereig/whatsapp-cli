// Package commands provides the CLI command implementations.
//
// # Dependency Injection (Future Work)
//
// The interfaces below document the dependencies of App.
// Currently App uses concrete types; these interfaces prepare for
// future testability improvements where mocks can be injected.
//
// To fully enable DI:
// 1. Align MediaDownloadRequest types between client and commands
// 2. Add StartSync to WAClient interface
// 3. Update App struct to use interfaces
// 4. Add constructor that accepts interfaces for testing
package commands

import (
	"context"
	"time"

	"github.com/vicentereig/whatsapp-cli/internal/store"
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
	DownloadMediaToFile(ctx context.Context, req MediaDownloadRequest, targetPath string) (int64, error)
}

// MediaDownloadRequest contains parameters for downloading media.
type MediaDownloadRequest struct {
	URL           string
	DirectPath    string
	MediaKey      []byte
	FileSHA256    []byte
	FileEncSHA256 []byte
	FileLength    uint64
	MediaType     string
	MimeType      string
}
