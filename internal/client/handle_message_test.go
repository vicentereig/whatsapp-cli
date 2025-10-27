package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	goproto "google.golang.org/protobuf/proto"
)

func TestHandleMessageReturnsTextContentWithoutMedia(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	msg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     types.NewJID("12345", types.DefaultUserServer),
				Sender:   types.NewJID("54321", types.DefaultUserServer),
				IsFromMe: false,
			},
			ID:        "txt-1",
			Timestamp: now,
		},
		Message: &proto.Message{
			Conversation: goproto.String("hello world"),
		},
	}

	details := HandleMessage(msg)

	assert.Equal(t, "txt-1", details.ID)
	assert.Equal(t, "12345@s.whatsapp.net", details.ChatJID)
	assert.Equal(t, "54321", details.Sender)
	assert.Equal(t, "hello world", details.Content)
	assert.Equal(t, now, details.Timestamp)
	assert.False(t, details.IsFromMe)
	assert.Nil(t, details.Media)
}

func TestHandleMessageExtractsImageMediaMetadata(t *testing.T) {
	now := time.Unix(1700000001, 0).UTC()
	directPath := "/v/t62.7119-24/ABC123"
	mediaKey := []byte{1, 2, 3}
	fileSha := []byte{4, 5, 6}
	fileEncSha := []byte{7, 8, 9}

	msg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     types.NewJID("67890", types.DefaultUserServer),
				Sender:   types.NewJID("09876", types.DefaultUserServer),
				IsFromMe: true,
			},
			ID:        "img-1",
			Timestamp: now,
		},
		Message: &proto.Message{
			ImageMessage: &proto.ImageMessage{
				Caption:       goproto.String("Look at this"),
				DirectPath:    goproto.String(directPath),
				Mimetype:      goproto.String("image/jpeg"),
				FileLength:    goproto.Uint64(2048),
				MediaKey:      mediaKey,
				FileSHA256:    fileSha,
				FileEncSHA256: fileEncSha,
			},
		},
	}

	details := HandleMessage(msg)
	require.NotNil(t, details.Media)

	assert.Equal(t, "img-1", details.ID)
	assert.Equal(t, "67890@s.whatsapp.net", details.ChatJID)
	assert.Equal(t, "Look at this", details.Content)
	assert.True(t, details.IsFromMe)

	media := details.Media
	assert.Equal(t, "image", media.Type)
	assert.Equal(t, "Look at this", media.Caption)
	assert.Equal(t, "image/jpeg", media.MimeType)
	assert.Equal(t, directPath, media.DirectPath)
	assert.Equal(t, mediaKey, media.MediaKey)
	assert.Equal(t, fileSha, media.FileSHA256)
	assert.Equal(t, fileEncSha, media.FileEncSHA256)
	assert.Equal(t, uint64(2048), media.FileLength)
}
