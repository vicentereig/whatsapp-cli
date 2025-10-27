package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Message struct {
	ID        string    `json:"id"`
	ChatJID   string    `json:"chat_jid"`
	ChatName  string    `json:"chat_name,omitempty"`
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	IsFromMe  bool      `json:"is_from_me"`
	MediaType string    `json:"media_type,omitempty"`
}

type Chat struct {
	JID             string    `json:"jid"`
	Name            string    `json:"name"`
	LastMessageTime time.Time `json:"last_message_time"`
	LastMessage     *string   `json:"last_message,omitempty"`
	LastSender      *string   `json:"last_sender,omitempty"`
	LastIsFromMe    *bool     `json:"last_is_from_me,omitempty"`
}

type Contact struct {
	PhoneNumber string `json:"phone_number"`
	Name        string `json:"name"`
	JID         string `json:"jid"`
}

type MessageStore struct {
	db *sql.DB
}

type MessageDownloadInfo struct {
	ID            string
	ChatJID       string
	ChatName      *string
	MediaType     string
	Filename      string
	DirectPath    string
	MimeType      string
	URL           string
	MediaKey      []byte
	FileSHA256    []byte
	FileEncSHA256 []byte
	FileLength    uint64
	LocalPath     *string
	DownloadedAt  *time.Time
	Sender        string
	Content       string
	MessageTime   time.Time
	IsFromMe      bool
}

type ListMessagesParams struct {
	After   *time.Time
	Before  *time.Time
	Sender  *string
	ChatJID *string
	Query   *string
	Limit   int
	Page    int
}

type ListChatsParams struct {
	Query *string
	Limit int
	Page  int
}

func NewMessageStore(dbPath string) (*MessageStore, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS chats (
			jid TEXT PRIMARY KEY,
			name TEXT,
			last_message_time TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS messages (
			id TEXT,
			chat_jid TEXT,
			sender TEXT,
			content TEXT,
			timestamp TIMESTAMP,
			is_from_me BOOLEAN,
			media_type TEXT,
			filename TEXT,
			url TEXT,
			direct_path TEXT,
			mime_type TEXT,
			media_key BLOB,
			file_sha256 BLOB,
			file_enc_sha256 BLOB,
			file_length INTEGER,
			local_path TEXT,
			downloaded_at TIMESTAMP,
			PRIMARY KEY (id, chat_jid),
			FOREIGN KEY (chat_jid) REFERENCES chats(jid)
		);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	if err := ensureMessageColumns(db); err != nil {
		db.Close()
		return nil, err
	}

	return &MessageStore{db: db}, nil
}

func ensureMessageColumns(db *sql.DB) error {
	required := map[string]string{
		"direct_path":   "TEXT",
		"mime_type":     "TEXT",
		"local_path":    "TEXT",
		"downloaded_at": "TIMESTAMP",
	}

	for column, columnType := range required {
		exists, err := columnExists(db, "messages", column)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE messages ADD COLUMN %s %s", column, columnType)); err != nil {
				// Ignore duplicate column errors for older SQLite versions that don't support IF NOT EXISTS.
				if !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
					return fmt.Errorf("failed to add column %s: %w", column, err)
				}
			}
		}
	}
	return nil
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("failed to inspect table %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return false, fmt.Errorf("failed to scan schema info: %w", err)
		}
		if strings.EqualFold(name, column) {
			return true, nil
		}
	}

	return false, nil
}

func (s *MessageStore) Close() error {
	return s.db.Close()
}

func (s *MessageStore) StoreChat(jid, name string, lastMessageTime time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO chats (jid, name, last_message_time) VALUES (?, ?, ?)
		ON CONFLICT(jid) DO UPDATE SET
			name = CASE
				WHEN excluded.name IS NOT NULL AND excluded.name != '' AND (excluded.name != chats.jid OR chats.name IS NULL OR chats.name = '' OR chats.name = chats.jid) THEN excluded.name
				WHEN chats.name IS NULL OR chats.name = '' THEN excluded.name
				ELSE chats.name
			END,
			last_message_time = excluded.last_message_time`,
		jid, name, lastMessageTime,
	)
	return err
}

func (s *MessageStore) StoreMessage(id, chatJID, sender, content string, timestamp time.Time, isFromMe bool,
	mediaType, filename, url, directPath, mimeType string, mediaKey, fileSHA256, fileEncSHA256 []byte, fileLength uint64) error {
	intFileLength := int64(0)
	if fileLength > 0 {
		intFileLength = int64(fileLength)
	}

	_, err := s.db.Exec(
		`INSERT INTO messages
		(id, chat_jid, sender, content, timestamp, is_from_me, media_type, filename, url, direct_path, mime_type, media_key, file_sha256, file_enc_sha256, file_length)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id, chat_jid) DO UPDATE SET
			sender = excluded.sender,
			content = excluded.content,
			timestamp = excluded.timestamp,
			is_from_me = excluded.is_from_me,
			media_type = excluded.media_type,
			filename = COALESCE(NULLIF(excluded.filename, ''), messages.filename),
			url = excluded.url,
			direct_path = COALESCE(NULLIF(excluded.direct_path, ''), messages.direct_path),
			mime_type = COALESCE(NULLIF(excluded.mime_type, ''), messages.mime_type),
			media_key = CASE WHEN excluded.media_key IS NOT NULL AND length(excluded.media_key) > 0 THEN excluded.media_key ELSE messages.media_key END,
			file_sha256 = CASE WHEN excluded.file_sha256 IS NOT NULL AND length(excluded.file_sha256) > 0 THEN excluded.file_sha256 ELSE messages.file_sha256 END,
			file_enc_sha256 = CASE WHEN excluded.file_enc_sha256 IS NOT NULL AND length(excluded.file_enc_sha256) > 0 THEN excluded.file_enc_sha256 ELSE messages.file_enc_sha256 END,
			file_length = CASE WHEN excluded.file_length > 0 THEN excluded.file_length ELSE messages.file_length END`,
		id, chatJID, sender, content, timestamp, isFromMe, mediaType, filename, url, directPath, mimeType, mediaKey, fileSHA256, fileEncSHA256, intFileLength,
	)
	return err
}

func (s *MessageStore) ListMessages(params ListMessagesParams) ([]Message, error) {
	query := `SELECT m.id, m.chat_jid, c.name, m.sender, m.content, m.timestamp, m.is_from_me, m.media_type
	          FROM messages m JOIN chats c ON m.chat_jid = c.jid WHERE 1=1`
	args := []interface{}{}

	if params.After != nil {
		query += " AND m.timestamp > ?"
		args = append(args, params.After)
	}
	if params.Before != nil {
		query += " AND m.timestamp < ?"
		args = append(args, params.Before)
	}
	if params.Sender != nil {
		query += " AND m.sender = ?"
		args = append(args, *params.Sender)
	}
	if params.ChatJID != nil {
		query += " AND m.chat_jid = ?"
		args = append(args, *params.ChatJID)
	}
	if params.Query != nil {
		query += " AND LOWER(m.content) LIKE LOWER(?)"
		args = append(args, "%"+*params.Query+"%")
	}

	query += " ORDER BY m.timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Page*params.Limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		err := rows.Scan(&m.ID, &m.ChatJID, &m.ChatName, &m.Sender, &m.Content, &m.Timestamp, &m.IsFromMe, &m.MediaType)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	return messages, nil
}

func (s *MessageStore) SearchContacts(query string) ([]Contact, error) {
	rows, err := s.db.Query(`
		SELECT jid, name FROM chats
		WHERE (LOWER(name) LIKE LOWER(?) OR LOWER(jid) LIKE LOWER(?))
		AND jid NOT LIKE '%@g.us'
		ORDER BY name LIMIT 50
	`, "%"+query+"%", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var c Contact
		var jid, name string
		if err := rows.Scan(&jid, &name); err != nil {
			return nil, err
		}
		c.JID = jid
		c.Name = name
		// Extract phone number from JID (before @)
		for idx := 0; idx < len(jid); idx++ {
			if jid[idx] == '@' {
				c.PhoneNumber = jid[:idx]
				break
			}
		}
		contacts = append(contacts, c)
	}

	return contacts, nil
}

func (s *MessageStore) GetMessageForDownload(id string, chatJID *string) (MessageDownloadInfo, error) {
	query := `
		SELECT
			m.id,
			m.chat_jid,
			c.name,
			m.sender,
			m.content,
			m.timestamp,
			m.is_from_me,
			m.media_type,
			m.filename,
			m.url,
			m.direct_path,
			m.mime_type,
			m.media_key,
			m.file_sha256,
			m.file_enc_sha256,
			COALESCE(m.file_length, 0),
			m.local_path,
			m.downloaded_at
		FROM messages m
		LEFT JOIN chats c ON m.chat_jid = c.jid
		WHERE m.id = ?`
	args := []interface{}{id}
	if chatJID != nil {
		query += " AND m.chat_jid = ?"
		args = append(args, *chatJID)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return MessageDownloadInfo{}, err
	}
	defer rows.Close()

	var infos []MessageDownloadInfo
	for rows.Next() {
		var info MessageDownloadInfo
		var localPath sql.NullString
		var downloadedAt sql.NullTime
		var fileLength sql.NullInt64

		if err := rows.Scan(
			&info.ID,
			&info.ChatJID,
			&info.ChatName,
			&info.Sender,
			&info.Content,
			&info.MessageTime,
			&info.IsFromMe,
			&info.MediaType,
			&info.Filename,
			&info.URL,
			&info.DirectPath,
			&info.MimeType,
			&info.MediaKey,
			&info.FileSHA256,
			&info.FileEncSHA256,
			&fileLength,
			&localPath,
			&downloadedAt,
		); err != nil {
			return MessageDownloadInfo{}, err
		}

		if fileLength.Valid && fileLength.Int64 > 0 {
			info.FileLength = uint64(fileLength.Int64)
		}
		if localPath.Valid {
			path := localPath.String
			info.LocalPath = &path
		}
		if downloadedAt.Valid {
			t := downloadedAt.Time
			info.DownloadedAt = &t
		}

		infos = append(infos, info)
	}

	if err := rows.Err(); err != nil {
		return MessageDownloadInfo{}, err
	}

	if len(infos) == 0 {
		return MessageDownloadInfo{}, sql.ErrNoRows
	}
	if len(infos) > 1 && chatJID == nil {
		return MessageDownloadInfo{}, fmt.Errorf("multiple messages found with ID %s; specify chat JID", id)
	}

	return infos[0], nil
}

func (s *MessageStore) MarkMediaDownloaded(id, chatJID, localPath string, downloadedAt time.Time) error {
	_, err := s.db.Exec(
		`UPDATE messages
		 SET local_path = ?, downloaded_at = ?
		 WHERE id = ? AND chat_jid = ?`,
		localPath, downloadedAt, id, chatJID,
	)
	return err
}

func (s *MessageStore) ListChats(params ListChatsParams) ([]Chat, error) {
	query := "SELECT jid, name, last_message_time FROM chats WHERE 1=1"
	args := []interface{}{}

	if params.Query != nil {
		query += " AND (LOWER(name) LIKE LOWER(?) OR jid LIKE ?)"
		args = append(args, "%"+*params.Query+"%", "%"+*params.Query+"%")
	}

	query += " ORDER BY last_message_time DESC LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Page*params.Limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var c Chat
		if err := rows.Scan(&c.JID, &c.Name, &c.LastMessageTime); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}

	return chats, nil
}
