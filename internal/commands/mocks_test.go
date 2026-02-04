package commands

import (
	"context"
	"time"

	"github.com/vicentereig/whatsapp-cli/internal/store"
	"github.com/vicentereig/whatsapp-cli/internal/types"
)

// MockMessageStore implements MessageStore for testing.
type MockMessageStore struct {
	ListMessagesFunc        func(params store.ListMessagesParams) ([]store.Message, error)
	SearchContactsFunc      func(query string) ([]store.Contact, error)
	ListChatsFunc           func(params store.ListChatsParams) ([]store.Chat, error)
	StoreChatFunc           func(jid, name string, lastMessageTime time.Time) error
	StoreMessageFunc        func(id, chatJID, sender, content string, timestamp time.Time, isFromMe bool, mediaType, filename, url, directPath, mimeType string, mediaKey, fileSHA256, fileEncSHA256 []byte, fileLength uint64) error
	GetMessageForDownloadFunc func(id string, chatJID *string) (store.MessageDownloadInfo, error)
	MarkMediaDownloadedFunc func(id, chatJID, localPath string, downloadedAt time.Time) error
	CloseFunc               func() error
}

func (m *MockMessageStore) ListMessages(params store.ListMessagesParams) ([]store.Message, error) {
	if m.ListMessagesFunc != nil {
		return m.ListMessagesFunc(params)
	}
	return nil, nil
}

func (m *MockMessageStore) SearchContacts(query string) ([]store.Contact, error) {
	if m.SearchContactsFunc != nil {
		return m.SearchContactsFunc(query)
	}
	return nil, nil
}

func (m *MockMessageStore) ListChats(params store.ListChatsParams) ([]store.Chat, error) {
	if m.ListChatsFunc != nil {
		return m.ListChatsFunc(params)
	}
	return nil, nil
}

func (m *MockMessageStore) StoreChat(jid, name string, lastMessageTime time.Time) error {
	if m.StoreChatFunc != nil {
		return m.StoreChatFunc(jid, name, lastMessageTime)
	}
	return nil
}

func (m *MockMessageStore) StoreMessage(id, chatJID, sender, content string, timestamp time.Time, isFromMe bool, mediaType, filename, url, directPath, mimeType string, mediaKey, fileSHA256, fileEncSHA256 []byte, fileLength uint64) error {
	if m.StoreMessageFunc != nil {
		return m.StoreMessageFunc(id, chatJID, sender, content, timestamp, isFromMe, mediaType, filename, url, directPath, mimeType, mediaKey, fileSHA256, fileEncSHA256, fileLength)
	}
	return nil
}

func (m *MockMessageStore) GetMessageForDownload(id string, chatJID *string) (store.MessageDownloadInfo, error) {
	if m.GetMessageForDownloadFunc != nil {
		return m.GetMessageForDownloadFunc(id, chatJID)
	}
	return store.MessageDownloadInfo{}, nil
}

func (m *MockMessageStore) MarkMediaDownloaded(id, chatJID, localPath string, downloadedAt time.Time) error {
	if m.MarkMediaDownloadedFunc != nil {
		return m.MarkMediaDownloadedFunc(id, chatJID, localPath, downloadedAt)
	}
	return nil
}

func (m *MockMessageStore) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// MockWAClient implements WAClient for testing.
type MockWAClient struct {
	IsAuthenticatedFunc     func() bool
	AuthenticateFunc        func(ctx context.Context) error
	ConnectFunc             func(ctx context.Context) error
	DisconnectFunc          func()
	SendMessageFunc         func(ctx context.Context, recipient, message string) error
	ResolveChatNameFunc     func(ctx context.Context, jid string, evt interface{}) string
	DownloadMediaToFileFunc func(ctx context.Context, req types.MediaDownloadRequest, targetPath string) (int64, error)
	StartSyncFunc           func(ctx context.Context, eventHandler func(interface{})) error
}

func (m *MockWAClient) IsAuthenticated() bool {
	if m.IsAuthenticatedFunc != nil {
		return m.IsAuthenticatedFunc()
	}
	return true
}

func (m *MockWAClient) Authenticate(ctx context.Context) error {
	if m.AuthenticateFunc != nil {
		return m.AuthenticateFunc(ctx)
	}
	return nil
}

func (m *MockWAClient) Connect(ctx context.Context) error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx)
	}
	return nil
}

func (m *MockWAClient) Disconnect() {
	if m.DisconnectFunc != nil {
		m.DisconnectFunc()
	}
}

func (m *MockWAClient) SendMessage(ctx context.Context, recipient, message string) error {
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(ctx, recipient, message)
	}
	return nil
}

func (m *MockWAClient) ResolveChatName(ctx context.Context, jid string, evt interface{}) string {
	if m.ResolveChatNameFunc != nil {
		return m.ResolveChatNameFunc(ctx, jid, evt)
	}
	return jid
}

func (m *MockWAClient) DownloadMediaToFile(ctx context.Context, req types.MediaDownloadRequest, targetPath string) (int64, error) {
	if m.DownloadMediaToFileFunc != nil {
		return m.DownloadMediaToFileFunc(ctx, req, targetPath)
	}
	return 0, nil
}

func (m *MockWAClient) StartSync(ctx context.Context, eventHandler func(interface{})) error {
	if m.StartSyncFunc != nil {
		return m.StartSyncFunc(ctx, eventHandler)
	}
	return nil
}
