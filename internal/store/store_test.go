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
	err = store.StoreMessage("msg1", chatJID, "1234", "Hello", time.Now(), false, "", "", "", "", "", nil, nil, nil, 0)
	assert.NoError(t, err)
}

func TestListMessages(t *testing.T) {
	store := setupTestDB(t)
	chatJID := "1234@s.whatsapp.net"

	// Setup test data
	store.StoreChat(chatJID, "John Doe", time.Now())
	now := time.Now()
	store.StoreMessage("msg1", chatJID, "1234", "Hello", now, false, "", "", "", "", "", nil, nil, nil, 0)
	store.StoreMessage("msg2", chatJID, "1234", "World", now.Add(time.Second), false, "", "", "", "", "", nil, nil, nil, 0)

	messages, err := store.ListMessages(ListMessagesParams{ChatJID: &chatJID, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "World", messages[0].Content) // Most recent first
	assert.Equal(t, "Hello", messages[1].Content)
}

func TestGetMessageForDownload(t *testing.T) {
	store := setupTestDB(t)
	chatJID := "1234@s.whatsapp.net"

	require.NoError(t, store.StoreChat(chatJID, "John Doe", time.Now()))

	now := time.Now().UTC().Truncate(time.Second)
	mediaKey := []byte{1, 2, 3}
	fileSHA := []byte{4, 5, 6}
	fileEncSHA := []byte{7, 8, 9}

	err := store.StoreMessage(
		"msg1",
		chatJID,
		"1234",
		"Sample caption",
		now,
		false,
		"image",
		"photo.jpg",
		"https://example.com/image",
		"/media/direct/path",
		"image/jpeg",
		mediaKey,
		fileSHA,
		fileEncSHA,
		1024,
	)
	require.NoError(t, err)

	info, err := store.GetMessageForDownload("msg1", nil)
	require.NoError(t, err)

	assert.Equal(t, "msg1", info.ID)
	assert.Equal(t, chatJID, info.ChatJID)
	assert.Equal(t, "image", info.MediaType)
	assert.Equal(t, "photo.jpg", info.Filename)
	assert.Equal(t, "/media/direct/path", info.DirectPath)
	assert.Equal(t, "image/jpeg", info.MimeType)
	assert.Equal(t, uint64(1024), info.FileLength)
	assert.Equal(t, mediaKey, info.MediaKey)
	assert.Equal(t, fileSHA, info.FileSHA256)
	assert.Equal(t, fileEncSHA, info.FileEncSHA256)
	assert.Nil(t, info.LocalPath)

	err = store.MarkMediaDownloaded("msg1", chatJID, "/tmp/photo.jpg", now.Add(time.Minute))
	require.NoError(t, err)

	infoAfter, err := store.GetMessageForDownload("msg1", nil)
	require.NoError(t, err)

	require.NotNil(t, infoAfter.LocalPath)
	assert.Equal(t, "/tmp/photo.jpg", *infoAfter.LocalPath)
	require.NotNil(t, infoAfter.DownloadedAt)
	assert.True(t, infoAfter.DownloadedAt.Equal(now.Add(time.Minute)))
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
