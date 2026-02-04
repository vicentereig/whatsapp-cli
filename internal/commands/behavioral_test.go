package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vicentereig/whatsapp-cli/internal/store"
)

// Helper to create string pointer
func ptr(s string) *string {
	return &s
}

// Response is the standard JSON response format
type Response struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *string         `json:"error"`
}

func parseResponse(t *testing.T, result string) Response {
	t.Helper()
	var resp Response
	err := json.Unmarshal([]byte(result), &resp)
	require.NoError(t, err, "response should be valid JSON: %s", result)
	return resp
}

// TestListMessages_FiltersByChat verifies that --chat flag filters messages correctly.
// This is the bug we fixed in PR #8.
func TestListMessages_FiltersByChat(t *testing.T) {
	targetJID := "target@s.whatsapp.net"
	otherJID := "other@s.whatsapp.net"

	allMessages := []store.Message{
		{ID: "1", ChatJID: targetJID, Content: "hello from target", Timestamp: time.Now()},
		{ID: "2", ChatJID: otherJID, Content: "hello from other", Timestamp: time.Now()},
		{ID: "3", ChatJID: targetJID, Content: "another from target", Timestamp: time.Now()},
	}

	mockStore := &MockMessageStore{
		ListMessagesFunc: func(params store.ListMessagesParams) ([]store.Message, error) {
			// Simulate filtering behavior
			if params.ChatJID != nil {
				var filtered []store.Message
				for _, m := range allMessages {
					if m.ChatJID == *params.ChatJID {
						filtered = append(filtered, m)
					}
				}
				return filtered, nil
			}
			return allMessages, nil
		},
	}

	app := NewAppWithDeps(&MockWAClient{}, mockStore, "/tmp", "test")

	// When: ListMessages called with chat filter
	result := app.ListMessages(ptr(targetJID), nil, 10, 0)

	// Then: Only messages from target chat are returned
	resp := parseResponse(t, result)
	require.True(t, resp.Success, "should succeed")

	var messages []store.Message
	err := json.Unmarshal(resp.Data, &messages)
	require.NoError(t, err)
	require.Len(t, messages, 2, "should return only 2 messages from target chat")

	for _, m := range messages {
		require.Equal(t, targetJID, m.ChatJID, "all messages should be from target chat")
	}
}

// TestListMessages_RespectsLimit verifies that --limit flag is honored.
func TestListMessages_RespectsLimit(t *testing.T) {
	allMessages := []store.Message{
		{ID: "1", Content: "msg1", Timestamp: time.Now()},
		{ID: "2", Content: "msg2", Timestamp: time.Now()},
		{ID: "3", Content: "msg3", Timestamp: time.Now()},
		{ID: "4", Content: "msg4", Timestamp: time.Now()},
		{ID: "5", Content: "msg5", Timestamp: time.Now()},
	}

	var capturedLimit int
	mockStore := &MockMessageStore{
		ListMessagesFunc: func(params store.ListMessagesParams) ([]store.Message, error) {
			capturedLimit = params.Limit
			// Simulate limit behavior
			if params.Limit > 0 && params.Limit < len(allMessages) {
				return allMessages[:params.Limit], nil
			}
			return allMessages, nil
		},
	}

	app := NewAppWithDeps(&MockWAClient{}, mockStore, "/tmp", "test")

	// When: ListMessages called with limit=2
	result := app.ListMessages(nil, nil, 2, 0)

	// Then: Only 2 messages returned and limit was passed to store
	resp := parseResponse(t, result)
	require.True(t, resp.Success)
	require.Equal(t, 2, capturedLimit, "limit should be passed to store")

	var messages []store.Message
	err := json.Unmarshal(resp.Data, &messages)
	require.NoError(t, err)
	require.Len(t, messages, 2, "should return only 2 messages")
}

// TestDownloadMedia_MissingMessage verifies proper error for non-existent message.
func TestDownloadMedia_MissingMessage(t *testing.T) {
	mockStore := &MockMessageStore{
		GetMessageForDownloadFunc: func(id string, chatJID *string) (store.MessageDownloadInfo, error) {
			return store.MessageDownloadInfo{}, sql.ErrNoRows
		},
	}

	app := NewAppWithDeps(&MockWAClient{}, mockStore, "/tmp", "test")

	// When: DownloadMedia called for non-existent message
	result := app.DownloadMedia(context.Background(), "nonexistent123", nil, "")

	// Then: Returns error with proper message
	resp := parseResponse(t, result)
	require.False(t, resp.Success, "should fail")
	require.NotNil(t, resp.Error)
	require.Contains(t, *resp.Error, "not found", "error should indicate message not found")
}

// TestDownloadMedia_EmptyMessageID verifies proper error for empty message ID.
func TestDownloadMedia_EmptyMessageID(t *testing.T) {
	app := NewAppWithDeps(&MockWAClient{}, &MockMessageStore{}, "/tmp", "test")

	// When: DownloadMedia called with empty message ID
	result := app.DownloadMedia(context.Background(), "", nil, "")

	// Then: Returns error
	resp := parseResponse(t, result)
	require.False(t, resp.Success, "should fail")
	require.NotNil(t, resp.Error)
	require.Contains(t, *resp.Error, "required", "error should indicate ID is required")
}

// TestDownloadMedia_NoMedia verifies proper error when message has no media.
func TestDownloadMedia_NoMedia(t *testing.T) {
	mockStore := &MockMessageStore{
		GetMessageForDownloadFunc: func(id string, chatJID *string) (store.MessageDownloadInfo, error) {
			// Return message with no media info
			return store.MessageDownloadInfo{
				ID:      id,
				ChatJID: "chat@jid",
				// No MediaType, DirectPath, or MediaKey
			}, nil
		},
	}

	app := NewAppWithDeps(&MockWAClient{}, mockStore, "/tmp", "test")

	// When: DownloadMedia called for message without media
	result := app.DownloadMedia(context.Background(), "textonly123", nil, "")

	// Then: Returns error indicating no media
	resp := parseResponse(t, result)
	require.False(t, resp.Success, "should fail")
	require.NotNil(t, resp.Error)
	require.Contains(t, *resp.Error, "no downloadable media", "error should indicate no media")
}

// TestSearchContacts_ReturnsResults verifies contact search works.
func TestSearchContacts_ReturnsResults(t *testing.T) {
	mockStore := &MockMessageStore{
		SearchContactsFunc: func(query string) ([]store.Contact, error) {
			if query == "john" {
				return []store.Contact{
					{Name: "John Doe", PhoneNumber: "1234567890", JID: "1234567890@s.whatsapp.net"},
				}, nil
			}
			return nil, nil
		},
	}

	app := NewAppWithDeps(&MockWAClient{}, mockStore, "/tmp", "test")

	// When: SearchContacts called with query
	result := app.SearchContacts("john")

	// Then: Returns matching contacts
	resp := parseResponse(t, result)
	require.True(t, resp.Success)

	var contacts []store.Contact
	err := json.Unmarshal(resp.Data, &contacts)
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	require.Equal(t, "John Doe", contacts[0].Name)
}

// TestListChats_ReturnsChats verifies chat listing works.
func TestListChats_ReturnsChats(t *testing.T) {
	mockStore := &MockMessageStore{
		ListChatsFunc: func(params store.ListChatsParams) ([]store.Chat, error) {
			return []store.Chat{
				{JID: "chat1@jid", Name: "Chat One", LastMessageTime: time.Now()},
				{JID: "chat2@jid", Name: "Chat Two", LastMessageTime: time.Now()},
			}, nil
		},
	}

	app := NewAppWithDeps(&MockWAClient{}, mockStore, "/tmp", "test")

	// When: ListChats called
	result := app.ListChats(nil, 10, 0)

	// Then: Returns chats
	resp := parseResponse(t, result)
	require.True(t, resp.Success)

	var chats []store.Chat
	err := json.Unmarshal(resp.Data, &chats)
	require.NoError(t, err)
	require.Len(t, chats, 2)
}
