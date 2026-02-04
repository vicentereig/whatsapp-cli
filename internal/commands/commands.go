package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vicentereig/whatsapp-cli/internal/client"
	"github.com/vicentereig/whatsapp-cli/internal/output"
	"github.com/vicentereig/whatsapp-cli/internal/store"
	"github.com/vicentereig/whatsapp-cli/internal/types"
	"go.mau.fi/whatsmeow/types/events"
)

type App struct {
	client          WAClient
	store           MessageStore
	version         string
	storeDir        string
	mediaDownloader func(ctx context.Context, info store.MessageDownloadInfo, targetPath string) (int64, error)
	mediaWorker     *mediaDownloadWorker
}

// NewApp creates a new App with production dependencies.
func NewApp(storeDir, version string) (*App, error) {
	cli, err := client.NewWAClient(storeDir)
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(storeDir, "messages.db")
	st, err := store.NewMessageStore(dbPath)
	if err != nil {
		return nil, err
	}

	app := &App{
		client:   cli,
		store:    st,
		version:  resolveVersion(version, gitDescribe),
		storeDir: storeDir,
	}
	app.mediaDownloader = app.downloadMediaWithClient
	return app, nil
}

// NewAppWithDeps creates a new App with injected dependencies for testing.
func NewAppWithDeps(client WAClient, store MessageStore, storeDir, version string) *App {
	app := &App{
		client:   client,
		store:    store,
		version:  version,
		storeDir: storeDir,
	}
	return app
}

func (a *App) Close() {
	if a.mediaWorker != nil {
		a.mediaWorker.Stop()
	}
	if a.client != nil {
		a.client.Disconnect()
	}
	if a.store != nil {
		a.store.Close()
	}
}

func (a *App) Auth(ctx context.Context) string {
	if a.client.IsAuthenticated() {
		return output.Success(map[string]interface{}{
			"authenticated": true,
			"message":       "Already authenticated",
		})
	}

	if err := a.client.Authenticate(ctx); err != nil {
		return output.Error(err)
	}

	return output.Success(map[string]interface{}{
		"authenticated": true,
		"message":       "Successfully authenticated",
	})
}

func (a *App) ListMessages(chatJID *string, query *string, limit, page int) string {
	messages, err := a.store.ListMessages(store.ListMessagesParams{
		ChatJID: chatJID,
		Query:   query,
		Limit:   limit,
		Page:    page,
	})
	if err != nil {
		return output.Error(err)
	}

	return output.Success(messages)
}

func (a *App) SearchContacts(query string) string {
	contacts, err := a.store.SearchContacts(query)
	if err != nil {
		return output.Error(err)
	}

	return output.Success(contacts)
}

func (a *App) ListChats(query *string, limit, page int) string {
	chats, err := a.store.ListChats(store.ListChatsParams{
		Query: query,
		Limit: limit,
		Page:  page,
	})
	if err != nil {
		return output.Error(err)
	}

	return output.Success(chats)
}

func (a *App) SendMessage(ctx context.Context, recipient, message string) string {
	if err := a.client.Connect(ctx); err != nil {
		return output.Error(err)
	}

	if err := a.client.SendMessage(ctx, recipient, message); err != nil {
		return output.Error(err)
	}

	// Store the message
	timestamp := time.Now()
	chatJID := recipient
	if !contains(recipient, "@") {
		chatJID = recipient + "@s.whatsapp.net"
	}

	// Resolve a friendly chat name when available (falls back to JID/recipient)
	chatName := a.client.ResolveChatName(ctx, chatJID, nil)
	if chatName == "" {
		chatName = recipient
	}

	// Store chat if needed
	a.store.StoreChat(chatJID, chatName, timestamp)
	a.store.StoreMessage(
		fmt.Sprintf("%d", timestamp.Unix()),
		chatJID,
		"me",
		message,
		timestamp,
		true,
		"", "", "", "", "",
		nil, nil, nil, 0,
	)

	return output.Success(map[string]interface{}{
		"sent":      true,
		"recipient": recipient,
		"message":   message,
	})
}

func (a *App) DownloadMedia(ctx context.Context, messageID string, chatJID *string, outputPath string) string {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return output.Error(fmt.Errorf("message ID is required"))
	}

	info, err := a.store.GetMessageForDownload(messageID, chatJID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return output.Error(fmt.Errorf("message %s not found", messageID))
		}
		return output.Error(err)
	}

	if strings.TrimSpace(info.MediaType) == "" || strings.TrimSpace(info.DirectPath) == "" || len(info.MediaKey) == 0 {
		return output.Error(fmt.Errorf("message %s has no downloadable media", messageID))
	}

	targetPath, bytesWritten, downloadedAt, err := a.downloadMediaAndPersist(ctx, info, outputPath)
	if err != nil {
		return output.Error(err)
	}

	response := map[string]interface{}{
		"message_id":    messageID,
		"chat_jid":      info.ChatJID,
		"path":          targetPath,
		"bytes":         bytesWritten,
		"media_type":    info.MediaType,
		"mime_type":     info.MimeType,
		"downloaded_at": downloadedAt.Format(time.RFC3339Nano),
	}
	if info.ChatName != nil && *info.ChatName != "" {
		response["chat_name"] = *info.ChatName
	}
	return output.Success(response)
}

func (a *App) resolveOutputPath(info store.MessageDownloadInfo, requested string) (string, error) {
	filename := sanitizeFilename(filenameFor(info))
	if filename == "" {
		filename = "file"
	}

	if strings.TrimSpace(requested) != "" {
		cleaned := requested
		if !filepath.IsAbs(cleaned) {
			if abs, err := filepath.Abs(cleaned); err == nil {
				cleaned = abs
			}
		}
		if info, err := os.Stat(cleaned); err == nil && info.IsDir() {
			return filepath.Join(cleaned, filename), nil
		}
		if strings.HasSuffix(cleaned, string(os.PathSeparator)) {
			return filepath.Join(cleaned, filename), nil
		}
		return cleaned, nil
	}

	baseDir := filepath.Join(a.storeDir, "media", sanitizeSegment(info.ChatJID), sanitizeSegment(info.ID))
	if info.MediaType != "" {
		baseDir = filepath.Join(baseDir, sanitizeSegment(info.MediaType))
	}
	if abs, err := filepath.Abs(baseDir); err == nil {
		baseDir = abs
	}
	return filepath.Join(baseDir, filename), nil
}

var pathReplacer = strings.NewReplacer(
	"/", "_",
	"\\", "_",
	":", "_",
	"@", "_",
	"?", "_",
	"*", "_",
	"<", "_",
	">", "_",
	"|", "_",
)

func sanitizeSegment(seg string) string {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return "unknown"
	}
	seg = pathReplacer.Replace(seg)
	seg = strings.ReplaceAll(seg, "..", "_")
	return seg
}

const maxFilenameLen = 200 // Leave room for directory path; most filesystems allow 255

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "file"
	}
	name = pathReplacer.Replace(name)
	name = strings.ReplaceAll(name, string(os.PathSeparator), "_")
	name = strings.ReplaceAll(name, "..", "_")
	// Truncate if too long (preserve extension if possible)
	if len(name) > maxFilenameLen {
		ext := filepath.Ext(name)
		if len(ext) < 20 && len(ext) > 0 {
			base := name[:maxFilenameLen-len(ext)]
			name = base + ext
		} else {
			name = name[:maxFilenameLen]
		}
	}
	return name
}

func filenameFor(info store.MessageDownloadInfo) string {
	if trimmed := strings.TrimSpace(info.Filename); trimmed != "" {
		return trimmed
	}
	if ext := extensionForMime(info.MimeType); ext != "" {
		return info.ID + ext
	}
	switch strings.ToLower(strings.TrimSpace(info.MediaType)) {
	case "image":
		return info.ID + ".jpg"
	case "video":
		return info.ID + ".mp4"
	case "audio":
		return info.ID + ".ogg"
	case "document":
		return info.ID
	default:
		return info.ID
	}
}

func extensionForMime(mimeType string) string {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if mimeType == "" {
		return ""
	}
	if exts, err := mime.ExtensionsByType(mimeType); err == nil {
		for _, ext := range exts {
			switch ext {
			case ".jpe":
				return ".jpg"
			default:
				if ext != "" {
					return ext
				}
			}
		}
	}
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "audio/ogg":
		return ".ogg"
	case "audio/mpeg":
		return ".mp3"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

func (a *App) downloadMediaWithClient(ctx context.Context, info store.MessageDownloadInfo, targetPath string) (int64, error) {
	if a.client == nil {
		return 0, fmt.Errorf("whatsapp client not initialized")
	}
	if err := a.client.Connect(ctx); err != nil {
		return 0, err
	}
	req := types.MediaDownloadRequest{
		DirectPath:    info.DirectPath,
		MediaKey:      info.MediaKey,
		FileSHA256:    info.FileSHA256,
		FileEncSHA256: info.FileEncSHA256,
		FileLength:    info.FileLength,
		MediaType:     info.MediaType,
		MimeType:      info.MimeType,
	}
	return a.client.DownloadMediaToFile(ctx, req, targetPath)
}

func (a *App) downloadMediaAndPersist(ctx context.Context, info store.MessageDownloadInfo, requestedPath string) (string, int64, time.Time, error) {
	finalPath, err := a.resolveOutputPath(info, requestedPath)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return "", 0, time.Time{}, fmt.Errorf("failed to create destination directory: %w", err)
	}

	downloader := a.mediaDownloader
	if downloader == nil {
		downloader = a.downloadMediaWithClient
	}

	bytesWritten, err := downloader(ctx, info, finalPath)
	if err != nil {
		return "", 0, time.Time{}, err
	}

	now := time.Now().UTC()
	if err := a.store.MarkMediaDownloaded(info.ID, info.ChatJID, finalPath, now); err != nil {
		return "", 0, time.Time{}, fmt.Errorf("failed to mark media downloaded: %w", err)
	}

	return finalPath, bytesWritten, now, nil
}

func (a *App) processMediaJob(ctx context.Context, job mediaJob) error {
	if a.store == nil {
		return fmt.Errorf("message store not initialized")
	}
	info, err := a.store.GetMessageForDownload(job.messageID, &job.chatJID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if strings.TrimSpace(info.DirectPath) == "" || len(info.MediaKey) == 0 {
		return nil
	}
	if info.LocalPath != nil {
		if _, err := os.Stat(*info.LocalPath); err == nil {
			return nil
		}
	}
	_, _, _, err = a.downloadMediaAndPersist(ctx, info, "")
	return err
}

type mediaJob struct {
	messageID string
	chatJID   string
}

type mediaDownloadWorker struct {
	app     *App
	workers int
	jobs    chan mediaJob
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// Error tracking
	mu             sync.Mutex
	expiredCount   int // 403/404/410 errors (media expired/deleted)
	otherErrors    int
	otherErrorMsgs []string // Keep first few for debugging
}

func newMediaDownloadWorker(app *App, workers int) *mediaDownloadWorker {
	if workers <= 0 {
		workers = 2
	}
	return &mediaDownloadWorker{
		app:     app,
		workers: workers,
		jobs:    make(chan mediaJob, workers*4),
	}
}

func (w *mediaDownloadWorker) Start(ctx context.Context) {
	if w == nil {
		return
	}
	w.ctx, w.cancel = context.WithCancel(ctx)
	for i := 0; i < w.workers; i++ {
		w.wg.Add(1)
		go w.run()
	}
}

func (w *mediaDownloadWorker) run() {
	defer w.wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			return
		case job := <-w.jobs:
			if err := w.app.processMediaJob(w.ctx, job); err != nil {
				w.trackError(err)
			}
		}
	}
}

func (w *mediaDownloadWorker) trackError(err error) {
	errStr := err.Error()
	// Check for expected expired/deleted media errors
	isExpired := contains(errStr, "status code 403") ||
		contains(errStr, "status code 404") ||
		contains(errStr, "status code 410")

	w.mu.Lock()
	defer w.mu.Unlock()

	if isExpired {
		w.expiredCount++
	} else {
		w.otherErrors++
		// Keep first 5 other errors for debugging
		if len(w.otherErrorMsgs) < 5 {
			w.otherErrorMsgs = append(w.otherErrorMsgs, errStr)
		}
	}
}

func (w *mediaDownloadWorker) PrintSummary() {
	if w == nil {
		return
	}
	w.mu.Lock()
	expiredCount := w.expiredCount
	otherErrors := w.otherErrors
	otherErrorMsgs := w.otherErrorMsgs
	w.mu.Unlock()

	if expiredCount > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  Skipped %d expired/deleted media files (normal for old messages)\n", expiredCount)
	}
	if otherErrors > 0 {
		fmt.Fprintf(os.Stderr, "⚠️  %d media downloads failed:\n", otherErrors)
		for _, msg := range otherErrorMsgs {
			fmt.Fprintf(os.Stderr, "   - %s\n", msg)
		}
		if otherErrors > len(otherErrorMsgs) {
			fmt.Fprintf(os.Stderr, "   ... and %d more\n", otherErrors-len(otherErrorMsgs))
		}
	}
}

func (w *mediaDownloadWorker) Enqueue(job mediaJob) {
	if w == nil || w.ctx == nil {
		return
	}
	select {
	case w.jobs <- job:
	case <-w.ctx.Done():
	default:
		go func() {
			select {
			case w.jobs <- job:
			case <-w.ctx.Done():
			}
		}()
	}
}

func (w *mediaDownloadWorker) Stop() {
	if w == nil {
		return
	}
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
}

func contains(s, substr string) bool {
	for i := 0; i < len(s); i++ {
		if i+len(substr) <= len(s) && s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Sync connects to WhatsApp and continuously syncs messages to the database
func (a *App) Sync(ctx context.Context) string {
	messageCount := 0

	version := a.version
	if strings.TrimSpace(version) == "" {
		version = "unknown"
	}
	fmt.Fprintf(os.Stderr, "ℹ️  whatsapp-cli version: %s\n", version)

	worker := newMediaDownloadWorker(a, 4)
	worker.Start(ctx)
	a.mediaWorker = worker
	defer func() {
		worker.Stop()
		worker.PrintSummary()
		if a.mediaWorker == worker {
			a.mediaWorker = nil
		}
	}()

	// Create event handler
	eventHandler := func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			// Extract message details
			details := client.HandleMessage(v)
			id := details.ID
			chatJID := details.ChatJID
			sender := details.Sender
			content := details.Content
			msgTime := details.Timestamp
			isFromMe := details.IsFromMe
			mediaType := ""
			filename := ""
			url := ""
			directPath := ""
			mimeType := ""
			var mediaKey, fileSHA256, fileEncSHA256 []byte
			var fileLength uint64

			if details.Media != nil {
				mediaType = details.Media.Type
				filename = details.Media.Filename
				url = details.Media.URL
				directPath = details.Media.DirectPath
				mimeType = details.Media.MimeType
				mediaKey = details.Media.MediaKey
				fileSHA256 = details.Media.FileSHA256
				fileEncSHA256 = details.Media.FileEncSHA256
				fileLength = details.Media.FileLength
			}

			chatName := a.client.ResolveChatName(ctx, chatJID, v)
			if chatName == "" && chatJID != "" {
				chatName = chatJID
			}

			// Store chat
			a.store.StoreChat(chatJID, chatName, msgTime)

			// Store message
			a.store.StoreMessage(
				id,
				chatJID,
				sender,
				content,
				msgTime,
				isFromMe,
				mediaType,
				filename,
				url,
				directPath,
				mimeType,
				mediaKey, fileSHA256, fileEncSHA256, fileLength,
			)

			if directPath != "" && len(mediaKey) > 0 {
				worker.Enqueue(mediaJob{messageID: id, chatJID: chatJID})
			}

			messageCount++
			fmt.Fprintf(os.Stderr, "\r💬 Synced %d messages...", messageCount)

		case *events.HistorySync:
			fmt.Fprintf(os.Stderr, "\n📜 Processing history sync (%d conversations)...\n", len(v.Data.Conversations))
			for _, conv := range v.Data.Conversations {
				chatJID := conv.GetID()
				chatName := conv.GetName()
				if chatName == "" {
					chatName = a.client.ResolveChatName(ctx, chatJID, nil)
					if chatName == "" {
						chatName = chatJID
					}
				}

				// Process messages in this conversation
				for _, msg := range conv.Messages {
					if msg.Message == nil {
						continue
					}

					histMsg := msg.Message
					msgID := histMsg.Key.GetID()
					sender := histMsg.Key.GetParticipant()
					if sender == "" {
						sender = histMsg.Key.GetRemoteJID()
					}
					isFromMe := histMsg.Key.GetFromMe()
					msgTimestamp := time.Unix(int64(histMsg.GetMessageTimestamp()), 0)

					// Extract content
					content := ""
					mediaType := ""
					filename := ""
					url := ""
					directPath := ""
					mimeType := ""
					var mediaKey, fileSHA256, fileEncSHA256 []byte
					var fileLength uint64

					switch {
					case histMsg.Message.GetConversation() != "":
						content = histMsg.Message.GetConversation()
					case histMsg.Message.GetExtendedTextMessage() != nil:
						extText := histMsg.Message.GetExtendedTextMessage()
						content = extText.GetText()
					case histMsg.Message.GetImageMessage() != nil:
						img := histMsg.Message.GetImageMessage()
						mediaType = "image"
						content = img.GetCaption()
						// Don't use caption as filename - it can be very long text
						url = img.GetURL()
						directPath = img.GetDirectPath()
						mimeType = img.GetMimetype()
						mediaKey = img.GetMediaKey()
						fileSHA256 = img.GetFileSHA256()
						fileEncSHA256 = img.GetFileEncSHA256()
						fileLength = img.GetFileLength()
					case histMsg.Message.GetVideoMessage() != nil:
						video := histMsg.Message.GetVideoMessage()
						mediaType = "video"
						content = video.GetCaption()
						// Don't use caption as filename - it can be very long text
						url = video.GetURL()
						directPath = video.GetDirectPath()
						mimeType = video.GetMimetype()
						mediaKey = video.GetMediaKey()
						fileSHA256 = video.GetFileSHA256()
						fileEncSHA256 = video.GetFileEncSHA256()
						fileLength = video.GetFileLength()
					case histMsg.Message.GetAudioMessage() != nil:
						audio := histMsg.Message.GetAudioMessage()
						mediaType = "audio"
						content = "[Audio]"
						url = audio.GetURL()
						directPath = audio.GetDirectPath()
						mimeType = audio.GetMimetype()
						mediaKey = audio.GetMediaKey()
						fileSHA256 = audio.GetFileSHA256()
						fileEncSHA256 = audio.GetFileEncSHA256()
						fileLength = audio.GetFileLength()
					case histMsg.Message.GetDocumentMessage() != nil:
						doc := histMsg.Message.GetDocumentMessage()
						mediaType = "document"
						content = doc.GetCaption()
						filename = doc.GetFileName()
						url = doc.GetURL()
						directPath = doc.GetDirectPath()
						mimeType = doc.GetMimetype()
						mediaKey = doc.GetMediaKey()
						fileSHA256 = doc.GetFileSHA256()
						fileEncSHA256 = doc.GetFileEncSHA256()
						fileLength = doc.GetFileLength()
					}

					// Store chat
					a.store.StoreChat(chatJID, chatName, msgTimestamp)

					// Store message
					a.store.StoreMessage(
						msgID,
						chatJID,
						sender,
						content,
						msgTimestamp,
						isFromMe,
						mediaType,
						filename,
						url,
						directPath,
						mimeType,
						mediaKey, fileSHA256, fileEncSHA256, fileLength,
					)

					if directPath != "" && len(mediaKey) > 0 {
						worker.Enqueue(mediaJob{messageID: msgID, chatJID: chatJID})
					}

					messageCount++
				}
			}
			fmt.Fprintf(os.Stderr, "\r💬 Synced %d messages...", messageCount)

		case *events.Connected:
			fmt.Fprintln(os.Stderr, "\n✓ Connected to WhatsApp")
			fmt.Fprintln(os.Stderr, "🔄 Listening for messages... (Press Ctrl+C to stop)")

		case *events.Disconnected:
			fmt.Fprintln(os.Stderr, "\n⚠ Disconnected from WhatsApp")
		}
	}

	// Start syncing
	fmt.Fprintln(os.Stderr, "🚀 Starting WhatsApp sync...")
	if err := a.client.StartSync(ctx, eventHandler); err != nil {
		return output.Error(err)
	}

	// Wait for context cancellation (Ctrl+C)
	<-ctx.Done()

	fmt.Fprintf(os.Stderr, "\n\n✓ Sync completed. Total messages synced: %d\n", messageCount)

	return output.Success(map[string]interface{}{
		"synced":         true,
		"messages_count": messageCount,
	})
}

func resolveVersion(version string, describeFn func() (string, error)) string {
	if strings.TrimSpace(version) != "" && version != "dev" {
		return version
	}

	if describeFn != nil {
		if gitVersion, err := describeFn(); err == nil && strings.TrimSpace(gitVersion) != "" {
			return gitVersion
		}
	}

	if strings.TrimSpace(version) == "" {
		return "unknown"
	}
	return version
}

func gitDescribe() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--dirty", "--always")
	cmd.Env = os.Environ()
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
