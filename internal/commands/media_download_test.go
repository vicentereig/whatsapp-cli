package commands

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vicente/whatsapp-cli/internal/output"
	"github.com/vicente/whatsapp-cli/internal/store"
)

type fakeDownloadStats struct {
	bytes int64
}

type fakeDownloader struct {
	called     bool
	request    store.MessageDownloadInfo
	targetPath string
	bytes      int64
	err        error
}

func (f *fakeDownloader) download(ctx context.Context, info store.MessageDownloadInfo, targetPath string) (fakeDownloadStats, error) {
	f.called = true
	f.request = info
	f.targetPath = targetPath
	return fakeDownloadStats{bytes: f.bytes}, f.err
}

func TestDownloadMediaUsesMetadataAndReturnsJSON(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "messages.db")
	st, err := store.NewMessageStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { st.Close() })

	chatJID := "1234@s.whatsapp.net"
	require.NoError(t, st.StoreChat(chatJID, "John Doe", time.Now()))

	now := time.Now()
	mediaKey := []byte{1, 2, 3}
	fileSHA := []byte{4, 5, 6}
	fileEncSHA := []byte{7, 8, 9}

	require.NoError(t, st.StoreMessage(
		"msg1",
		chatJID,
		"1234",
		"Sample caption",
		now,
		false,
		"image",
		"photo.jpg",
		"https://example.com",
		"/media/direct/path",
		"image/jpeg",
		mediaKey,
		fileSHA,
		fileEncSHA,
		1024,
	))

	app := &App{
		store:    st,
		version:  "test",
		storeDir: tmpDir,
	}

	fake := &fakeDownloader{bytes: 1024}
	app.mediaDownloader = func(ctx context.Context, info store.MessageDownloadInfo, targetPath string) (int64, error) {
		stats, err := fake.download(ctx, info, targetPath)
		return stats.bytes, err
	}

	outputPath := filepath.Join(tmpDir, "downloaded.jpg")
	resJSON := app.DownloadMedia(context.Background(), "msg1", nil, outputPath)

	var res output.Result
	require.NoError(t, json.Unmarshal([]byte(resJSON), &res))

	assert.True(t, res.Success, "expected success JSON")
	assert.Nil(t, res.Error)

	dataMap, ok := res.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "msg1", dataMap["message_id"])
	assert.Equal(t, outputPath, dataMap["path"])
	assert.EqualValues(t, 1024, dataMap["bytes"])

	require.True(t, fake.called, "expected downloader to be invoked")
	assert.Equal(t, "/media/direct/path", fake.request.DirectPath)
	assert.Equal(t, outputPath, fake.targetPath)
	assert.Equal(t, chatJID, fake.request.ChatJID)
	assert.Equal(t, mediaKey, fake.request.MediaKey)
	assert.Equal(t, fileSHA, fake.request.FileSHA256)
	assert.Equal(t, fileEncSHA, fake.request.FileEncSHA256)

	infoAfter, err := st.GetMessageForDownload("msg1", nil)
	require.NoError(t, err)
	require.NotNil(t, infoAfter.LocalPath)
	assert.Equal(t, outputPath, *infoAfter.LocalPath)
}

func TestDownloadMediaErrorsWhenMetadataMissing(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "messages.db")
	st, err := store.NewMessageStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { st.Close() })

	chatJID := "123@s.whatsapp.net"
	require.NoError(t, st.StoreChat(chatJID, "Jane", time.Now()))
	require.NoError(t, st.StoreMessage(
		"msg2",
		chatJID,
		"123",
		"No media here",
		time.Now(),
		false,
		"text",
		"",
		"",
		"",
		"",
		nil,
		nil,
		nil,
		0,
	))

	app := &App{
		store:    st,
		version:  "test",
		storeDir: tmpDir,
		mediaDownloader: func(ctx context.Context, info store.MessageDownloadInfo, targetPath string) (int64, error) {
			return 0, nil
		},
	}

	resJSON := app.DownloadMedia(context.Background(), "msg2", nil, "")

	var res output.Result
	require.NoError(t, json.Unmarshal([]byte(resJSON), &res))

	assert.False(t, res.Success)
	require.NotNil(t, res.Error)
	assert.Contains(t, *res.Error, "no downloadable media")
}
