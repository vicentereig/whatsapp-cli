package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
			media_key BLOB,
			file_sha256 BLOB,
			file_enc_sha256 BLOB,
			file_length INTEGER,
			PRIMARY KEY (id, chat_jid),
			FOREIGN KEY (chat_jid) REFERENCES chats(jid)
		);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return &MessageStore{db: db}, nil
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
	mediaType, filename, url string, mediaKey, fileSHA256, fileEncSHA256 []byte, fileLength uint64) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO messages
		(id, chat_jid, sender, content, timestamp, is_from_me, media_type, filename, url, media_key, file_sha256, file_enc_sha256, file_length)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, chatJID, sender, content, timestamp, isFromMe, mediaType, filename, url, mediaKey, fileSHA256, fileEncSHA256, fileLength,
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
