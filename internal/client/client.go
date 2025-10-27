package client

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

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

// Helper to handle incoming messages
func HandleMessage(msg *events.Message) (id, chatJID, sender, content string, timestamp int64, isFromMe bool, mediaType, filename, url string) {
	id = msg.Info.ID
	chatJID = msg.Info.Chat.String()
	sender = msg.Info.Sender.User
	timestamp = msg.Info.Timestamp.Unix()
	isFromMe = msg.Info.IsFromMe

	if msg.Message != nil {
		if text := msg.Message.GetConversation(); text != "" {
			content = text
		} else if extText := msg.Message.GetExtendedTextMessage(); extText != nil {
			content = extText.GetText()
		} else if img := msg.Message.GetImageMessage(); img != nil {
			mediaType = "image"
			filename = img.GetCaption()
			url = img.GetURL()
			content = img.GetCaption()
		} else if video := msg.Message.GetVideoMessage(); video != nil {
			mediaType = "video"
			filename = video.GetCaption()
			url = video.GetURL()
			content = video.GetCaption()
		} else if audio := msg.Message.GetAudioMessage(); audio != nil {
			mediaType = "audio"
			url = audio.GetURL()
			content = "[Audio]"
		} else if doc := msg.Message.GetDocumentMessage(); doc != nil {
			mediaType = "document"
			filename = doc.GetFileName()
			url = doc.GetURL()
			content = doc.GetCaption()
		}
	}

	return
}
