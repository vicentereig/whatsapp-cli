package client

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type WAClient struct {
	client          *whatsmeow.Client
	storeDir        string
	eventHandler    func(interface{})
	contactLookup   func(ctx context.Context, user types.JID) (types.ContactInfo, error)
	groupInfoLookup func(ctx context.Context, jid types.JID) (*types.GroupInfo, error)
}

type MediaInfo struct {
	Type          string
	Filename      string
	URL           string
	DirectPath    string
	MimeType      string
	Caption       string
	MediaKey      []byte
	FileSHA256    []byte
	FileEncSHA256 []byte
	FileLength    uint64
}

type MessageDetails struct {
	ID        string
	ChatJID   string
	Sender    string
	Content   string
	Timestamp time.Time
	IsFromMe  bool
	Media     *MediaInfo
}

type MediaDownloadRequest struct {
	DirectPath    string
	MediaKey      []byte
	FileSHA256    []byte
	FileEncSHA256 []byte
	FileLength    uint64
	MediaType     string
	MimeType      string
}

func NewWAClient(storeDir string) (*WAClient, error) {
	// Create store directory
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %v", err)
	}

	dbLog := waLog.Stdout("Database", "ERROR", true)
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", fmt.Sprintf("file:%s/whatsapp.db?_foreign_keys=on", storeDir), dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			deviceStore = container.NewDevice()
		} else {
			return nil, fmt.Errorf("failed to get device: %v", err)
		}
	}

	logger := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, logger)

	return &WAClient{
		client:          client,
		storeDir:        storeDir,
		contactLookup:   contactLookupFunc(client),
		groupInfoLookup: groupInfoLookupFunc(client),
	}, nil
}

func (w *WAClient) IsAuthenticated() bool {
	return w.client.Store.ID != nil
}

func (w *WAClient) Authenticate(ctx context.Context) error {
	if w.IsAuthenticated() {
		return nil
	}

	qrChan, _ := w.client.GetQRChannel(ctx)
	if err := w.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	for evt := range qrChan {
		if evt.Event == "code" {
			fmt.Println("\nScan this QR code with your WhatsApp app:")
			// Use Medium error correction and compact output
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.M, os.Stdout)
		} else if evt.Event == "success" {
			fmt.Println("\n✓ Successfully authenticated!")
			return nil
		}
	}

	return fmt.Errorf("authentication failed")
}

func (w *WAClient) Connect(ctx context.Context) error {
	if !w.IsAuthenticated() {
		return w.Authenticate(ctx)
	}

	if err := w.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	return nil
}

func (w *WAClient) Disconnect() {
	if w.client != nil {
		w.client.Disconnect()
	}
}

func (w *WAClient) SendMessage(ctx context.Context, recipient, message string) error {
	if !w.client.IsConnected() {
		return fmt.Errorf("not connected to WhatsApp")
	}

	recipientJID, err := parseJID(recipient)
	if err != nil {
		return err
	}

	msg := &waProto.Message{
		Conversation: proto.String(message),
	}

	_, err = w.client.SendMessage(ctx, recipientJID, msg)
	return err
}

func (w *WAClient) AddEventHandler(handler func(interface{})) {
	w.client.AddEventHandler(handler)
}

func contactLookupFunc(cli *whatsmeow.Client) func(ctx context.Context, user types.JID) (types.ContactInfo, error) {
	if cli == nil || cli.Store == nil || cli.Store.Contacts == nil {
		return nil
	}
	return func(ctx context.Context, user types.JID) (types.ContactInfo, error) {
		return cli.Store.Contacts.GetContact(ctx, user)
	}
}

func groupInfoLookupFunc(cli *whatsmeow.Client) func(ctx context.Context, jid types.JID) (*types.GroupInfo, error) {
	if cli == nil {
		return nil
	}
	return func(ctx context.Context, jid types.JID) (*types.GroupInfo, error) {
		// whatsmeow's GetGroupInfo does not accept a context, so we ignore ctx here.
		info, err := cli.GetGroupInfo(jid)
		return info, err
	}
}

func bestContactName(info types.ContactInfo) string {
	if !info.Found {
		return ""
	}
	if name := strings.TrimSpace(info.FullName); name != "" {
		return name
	}
	if name := strings.TrimSpace(info.FirstName); name != "" {
		return name
	}
	if name := strings.TrimSpace(info.BusinessName); name != "" {
		return name
	}
	if name := strings.TrimSpace(info.PushName); name != "" && name != "-" {
		return name
	}
	if name := strings.TrimSpace(info.RedactedPhone); name != "" {
		return name
	}
	return ""
}

func (w *WAClient) ResolveChatName(ctx context.Context, chatJID string, msg *events.Message) string {
	if chatJID == "" && msg != nil {
		chatJID = msg.Info.Chat.String()
	}
	fallback := chatJID

	parsed, err := types.ParseJID(chatJID)
	if err == nil {
		// Group chats
		if parsed.Server == types.GroupServer || parsed.IsBroadcastList() {
			if w.groupInfoLookup != nil {
				if info, err := w.groupInfoLookup(ctx, parsed); err == nil && info != nil {
					if name := strings.TrimSpace(info.GroupName.Name); name != "" {
						return name
					}
				}
			}
		} else {
			if w.contactLookup != nil {
				if info, err := w.contactLookup(ctx, parsed.ToNonAD()); err == nil {
					if name := bestContactName(info); name != "" {
						return name
					}
				}
			}
		}
	}

	if msg != nil {
		if name := strings.TrimSpace(msg.Info.PushName); name != "" && name != "-" {
			return name
		}
	}

	return fallback
}

// StartSync connects to WhatsApp and registers event handlers for syncing messages
func (w *WAClient) StartSync(ctx context.Context, eventHandler func(interface{})) error {
	// Add event handler before connecting
	w.client.AddEventHandler(eventHandler)

	// Connect to WhatsApp
	if err := w.Connect(ctx); err != nil {
		return err
	}

	return nil
}

func parseJID(recipient string) (types.JID, error) {
	// If already a JID, parse it
	if contains(recipient, "@") {
		return types.ParseJID(recipient)
	}

	// Otherwise, assume it's a phone number
	return types.JID{
		User:   recipient,
		Server: "s.whatsapp.net",
	}, nil
}

func contains(s, substr string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == substr[0] {
			return true
		}
	}
	return false
}

func (w *WAClient) DownloadMediaToFile(ctx context.Context, req MediaDownloadRequest, targetPath string) (int64, error) {
	if w == nil || w.client == nil {
		return 0, fmt.Errorf("whatsapp client is not initialized")
	}
	if strings.TrimSpace(req.DirectPath) == "" {
		return 0, fmt.Errorf("media direct path is empty")
	}
	mediaType, err := mediaTypeFromString(req.MediaType)
	if err != nil {
		return 0, err
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(targetPath), ".wa-download-*")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	success := false
	defer func() {
		tmpFile.Close()
		if !success {
			os.Remove(tmpName)
		}
	}()

	length := -1
	if req.FileLength > 0 && req.FileLength < math.MaxInt32 {
		length = int(req.FileLength)
	}

	if err := w.client.DownloadMediaWithPathToFile(ctx, req.DirectPath, req.FileEncSHA256, req.FileSHA256, req.MediaKey, length, mediaType, "", tmpFile); err != nil {
		return 0, err
	}

	if err := tmpFile.Sync(); err != nil {
		return 0, fmt.Errorf("failed to flush media file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return 0, fmt.Errorf("failed to close media file: %w", err)
	}
	if err := os.Rename(tmpName, targetPath); err != nil {
		return 0, fmt.Errorf("failed to move downloaded media: %w", err)
	}
	success = true

	info, err := os.Stat(targetPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat downloaded media: %w", err)
	}
	return info.Size(), nil
}

func mediaTypeFromString(mediaType string) (whatsmeow.MediaType, error) {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image":
		return whatsmeow.MediaImage, nil
	case "video":
		return whatsmeow.MediaVideo, nil
	case "audio":
		return whatsmeow.MediaAudio, nil
	case "document":
		return whatsmeow.MediaDocument, nil
	case "sticker":
		return whatsmeow.MediaImage, nil
	default:
		return "", fmt.Errorf("unsupported media type: %s", mediaType)
	}
}

// Helper to handle incoming messages
func HandleMessage(msg *events.Message) MessageDetails {
	sender := msg.Info.Sender.User
	if sender == "" {
		if s := msg.Info.Sender.String(); s != "" {
			sender = s
		}
	}

	details := MessageDetails{
		ID:        msg.Info.ID,
		ChatJID:   msg.Info.Chat.String(),
		Sender:    sender,
		Timestamp: msg.Info.Timestamp,
		IsFromMe:  msg.Info.IsFromMe,
	}

	if msg.Message != nil {
		switch {
		case msg.Message.GetConversation() != "":
			details.Content = msg.Message.GetConversation()
		case msg.Message.GetExtendedTextMessage() != nil:
			details.Content = msg.Message.GetExtendedTextMessage().GetText()
		}

		if img := msg.Message.GetImageMessage(); img != nil {
			if details.Content == "" {
				details.Content = img.GetCaption()
			}
			details.Media = &MediaInfo{
				Type:          "image",
				Filename:      "",
				URL:           img.GetURL(),
				DirectPath:    img.GetDirectPath(),
				MimeType:      img.GetMimetype(),
				Caption:       img.GetCaption(),
				MediaKey:      cloneBytes(img.GetMediaKey()),
				FileSHA256:    cloneBytes(img.GetFileSHA256()),
				FileEncSHA256: cloneBytes(img.GetFileEncSHA256()),
				FileLength:    img.GetFileLength(),
			}
		} else if video := msg.Message.GetVideoMessage(); video != nil {
			if details.Content == "" {
				details.Content = video.GetCaption()
			}
			details.Media = &MediaInfo{
				Type:          "video",
				Filename:      "",
				URL:           video.GetURL(),
				DirectPath:    video.GetDirectPath(),
				MimeType:      video.GetMimetype(),
				Caption:       video.GetCaption(),
				MediaKey:      cloneBytes(video.GetMediaKey()),
				FileSHA256:    cloneBytes(video.GetFileSHA256()),
				FileEncSHA256: cloneBytes(video.GetFileEncSHA256()),
				FileLength:    video.GetFileLength(),
			}
		} else if audio := msg.Message.GetAudioMessage(); audio != nil {
			if details.Content == "" {
				details.Content = "[Audio]"
			}
			details.Media = &MediaInfo{
				Type:          "audio",
				Filename:      "",
				URL:           audio.GetURL(),
				DirectPath:    audio.GetDirectPath(),
				MimeType:      audio.GetMimetype(),
				Caption:       details.Content,
				MediaKey:      cloneBytes(audio.GetMediaKey()),
				FileSHA256:    cloneBytes(audio.GetFileSHA256()),
				FileEncSHA256: cloneBytes(audio.GetFileEncSHA256()),
				FileLength:    audio.GetFileLength(),
			}
		} else if doc := msg.Message.GetDocumentMessage(); doc != nil {
			if details.Content == "" {
				details.Content = doc.GetCaption()
			}
			filename := doc.GetFileName()
			details.Media = &MediaInfo{
				Type:          "document",
				Filename:      filename,
				URL:           doc.GetURL(),
				DirectPath:    doc.GetDirectPath(),
				MimeType:      doc.GetMimetype(),
				Caption:       doc.GetCaption(),
				MediaKey:      cloneBytes(doc.GetMediaKey()),
				FileSHA256:    cloneBytes(doc.GetFileSHA256()),
				FileEncSHA256: cloneBytes(doc.GetFileEncSHA256()),
				FileLength:    doc.GetFileLength(),
			}
		}
	}

	return details
}

func cloneBytes(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
