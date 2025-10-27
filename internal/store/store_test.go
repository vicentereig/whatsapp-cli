package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *MessageStore {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMessageStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	return store
}

func TestNewMessageStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMessageStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Verify database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestStoreChat(t *testing.T) {
	store := setupTestDB(t)

	err := store.StoreChat("1234@s.whatsapp.net", "John Doe", time.Now())
	assert.NoError(t, err)
}

func TestStoreChatDoesNotOverwriteFriendlyWithJID(t *testing.T) {
	store := setupTestDB(t)
	jid := "1234@s.whatsapp.net"

	require.NoError(t, store.StoreChat(jid, "John Doe", time.Now()))
	require.NoError(t, store.StoreChat(jid, jid, time.Now().Add(time.Minute)))

	chats, err := store.ListChats(ListChatsParams{Limit: 1})
	require.NoError(t, err)
	require.NotEmpty(t, chats)
	assert.Equal(t, "John Doe", chats[0].Name)
}

func TestStoreChatUpgradesNameFromJID(t *testing.T) {
	store := setupTestDB(t)
	jid := "5678@s.whatsapp.net"

	require.NoError(t, store.StoreChat(jid, jid, time.Now()))
	require.NoError(t, store.StoreChat(jid, "Jane Smith", time.Now().Add(time.Minute)))

	chats, err := store.ListChats(ListChatsParams{Limit: 1})
	require.NoError(t, err)
	require.NotEmpty(t, chats)
	assert.Equal(t, "Jane Smith", chats[0].Name)
}

func TestStoreMessage(t *testing.T) {
	store := setupTestDB(t)

	// First store a chat
	chatJID := "1234@s.whatsapp.net"
	err := store.StoreChat(chatJID, "John Doe", time.Now())
	require.NoError(t, err)

	// Then store a message
	err = store.StoreMessage("msg1", chatJID, "1234", "Hello", time.Now(), false, "", "", "", nil, nil, nil, 0)
	assert.NoError(t, err)
}

func TestListMessages(t *testing.T) {
	store := setupTestDB(t)
	chatJID := "1234@s.whatsapp.net"

	// Setup test data
	store.StoreChat(chatJID, "John Doe", time.Now())
	now := time.Now()
	store.StoreMessage("msg1", chatJID, "1234", "Hello", now, false, "", "", "", nil, nil, nil, 0)
	store.StoreMessage("msg2", chatJID, "1234", "World", now.Add(time.Second), false, "", "", "", nil, nil, nil, 0)

	messages, err := store.ListMessages(ListMessagesParams{ChatJID: &chatJID, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "World", messages[0].Content) // Most recent first
	assert.Equal(t, "Hello", messages[1].Content)
}

func TestSearchContacts(t *testing.T) {
	store := setupTestDB(t)

	// Setup test data
	store.StoreChat("1234@s.whatsapp.net", "John Doe", time.Now())
	store.StoreChat("5678@s.whatsapp.net", "Jane Smith", time.Now())
	store.StoreChat("9999@g.us", "Group Chat", time.Now()) // Should be excluded

	contacts, err := store.SearchContacts("John")
	require.NoError(t, err)
	assert.Len(t, contacts, 1)
	assert.Equal(t, "John Doe", contacts[0].Name)
}

func TestListChats(t *testing.T) {
	store := setupTestDB(t)

	// Setup test data
	store.StoreChat("1234@s.whatsapp.net", "John Doe", time.Now())
	store.StoreChat("5678@s.whatsapp.net", "Jane Smith", time.Now().Add(-time.Hour))

	chats, err := store.ListChats(ListChatsParams{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, chats, 2)
	assert.Equal(t, "John Doe", chats[0].Name) // Most recent first
}
