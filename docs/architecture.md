# WhatsApp CLI Architecture

## Overview

WhatsApp CLI is a standalone Go program built on `whatsmeow` for WhatsApp access from the command line. Codex, Claude Code, and other automation tools can use it.

## Architecture

```
┌─────────────────────┐
│   CLI Tool (Go)     │
│                     │
│  • whatsmeow lib    │ ← Direct WhatsApp Web API connection
│  • SQLite storage   │ ← Local message/session storage
│  • Cobra CLI        │ ← Command parsing (optional)
└─────────────────────┘
```

## Key Advantages

- **Single binary**: no Python or Node.js dependency. Compile and run it.
- **Direct access**: no HTTP middleware. The CLI queries WhatsApp directly.
- **Reuse code**: you can adapt most of the logic from the `whatsapp-mcp` bridge.
- **JSON output**: Codex and Claude Code can parse it directly.
- **Same authentication**: a QR code on first run, then a persistent session.

## Project Structure

```
whatsapp-cli/
├── main.go           # CLI entry point
├── cmd/              # Command implementations
│   ├── auth.go      # QR code authentication
│   ├── messages.go  # Read/search messages
│   ├── send.go      # Send messages/files
│   ├── contacts.go  # Search contacts
│   └── media.go     # Download media
├── store/           # SQLite databases
│   ├── whatsapp.db  # Session data (whatsmeow storage)
│   └── messages.db  # Message history
├── docs/            # Documentation
└── go.mod
```

## Core Commands

### Authentication
```bash
whatsapp-cli auth               # Show QR code, establish session
whatsapp-cli auth status        # Check authentication status
```

### Read Messages
```bash
whatsapp-cli messages list --chat JID --limit 20
whatsapp-cli messages search --query "meeting" --after 2025-01-01
whatsapp-cli messages get --id MESSAGE_ID --chat JID
whatsapp-cli messages context --id MESSAGE_ID --before 5 --after 5
```

### Send
```bash
whatsapp-cli send text --to PHONE_OR_JID --message "Hello"
whatsapp-cli send file --to PHONE_OR_JID --file /path/to/file.jpg
whatsapp-cli send audio --to PHONE_OR_JID --file /path/to/voice.ogg
```

### Contacts & Chats
```bash
whatsapp-cli contacts search --query "John"
whatsapp-cli chats list --limit 10
whatsapp-cli chats get --jid CHAT_JID
```

### Media
```bash
whatsapp-cli media download --message-id ID --chat-jid JID
```

## Output Format

All commands print output as JSON:

```json
{
  "success": true,
  "data": [...],
  "error": null
}
```

Example message output:
```json
{
  "success": true,
  "data": [
    {
      "id": "msg123",
      "chat_jid": "1234567890@s.whatsapp.net",
      "chat_name": "John Doe",
      "sender": "1234567890",
      "content": "Hello there!",
      "timestamp": "2025-10-26T10:30:00Z",
      "is_from_me": false,
      "media_type": null
    }
  ],
  "error": null
}
```

## Authentication Flow

### First-Time Login (No Session Exists)

Based on existing `whatsapp-mcp` code (`main.go:860-887`):

```go
if client.Store.ID == nil {
    // No ID stored, need to pair with phone
    qrChan, _ := client.GetQRChannel(context.Background())
    err = client.Connect()

    // Print QR code for pairing with phone
    for evt := range qrChan {
        if evt.Event == "code" {
            fmt.Println("\nScan this QR code with your WhatsApp app:")
            qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
        } else if evt.Event == "success" {
            // Connected! Session now saved to SQLite
            break
        }
    }
}
```

**What happens:**
1. The CLI detects that no session exists in `store/whatsapp.db`.
2. It generates a QR code and shows it in the terminal.
3. The user scans it with the WhatsApp mobile app (Settings → Linked Devices → Link a Device).
4. WhatsApp sends session credentials through the QR code flow.
5. `whatsmeow` saves the session data to SQLite automatically.
6. Done. The session lasts about 20 days.

### Subsequent Logins (Session Exists)

```go
else {
    // Already logged in, just connect
    err = client.Connect()
    if err != nil {
        logger.Errorf("Failed to connect: %v", err)
        return
    }
}
```

**What happens:**
1. The CLI reads the session from `store/whatsapp.db`.
2. It reconnects to WhatsApp servers automatically.
3. It does not need a QR code.
4. It works until the session expires, about 20 days later.

### Authentication Strategy

**Hybrid approach (recommended):**
- If a session exists, CLI commands connect automatically.
- If no session exists, the CLI shows the QR code inline and waits.
- An optional `--non-interactive` flag supports scripts. It fails if the user is not authenticated.

**Example flow:**
```bash
# First time
$ whatsapp-cli contacts search "John"
⚠ Not authenticated. Scan QR code:
[QR CODE APPEARS]
✓ Authenticated successfully!
[Shows results]

# Next time
$ whatsapp-cli contacts search "John"
[Shows results immediately]
```

## Session Storage

**The session lives in SQLite** (`store/whatsapp.db`) and holds:
- Device ID
- Encryption keys
- Registration info
- Contact data

**Message history** is stored in `store/messages.db`:
- Chats table (JID, name, last message time)
- Messages table (ID, content, sender, timestamp, media metadata)
- Media files stored on-demand in `store/{chat_jid}/`

**Location options:**
1. Current directory: `./store/` (default)
2. Home directory: `~/.whatsapp-cli/`
3. Custom path: `--db-path /path/to/store/`

## Implementation Steps

### 1. Create new Go module
```bash
cd whatsapp-cli
go mod init github.com/vicentereig/whatsapp-cli
go get go.mau.fi/whatsmeow
go get github.com/mdp/qrterminal
go get github.com/mattn/go-sqlite3
```

### 2. Copy auth + storage logic
- Reuse authentication code from `whatsapp-mcp/whatsapp-bridge/main.go` (lines 789-923)
- Reuse message store logic (lines 44-173)
- Reuse message handling (lines 411-471)

### 3. Implement CLI commands
- Create command structure (using Cobra or manual flag parsing)
- Implement JSON output formatters
- Add context-aware message retrieval
- Support all operations: read, send, search, download

### 4. Build as a single binary
```bash
go build -o whatsapp-cli
```

### 5. Optional: Cross-platform builds
```bash
GOOS=linux GOARCH=amd64 go build -o whatsapp-cli-linux
GOOS=darwin GOARCH=arm64 go build -o whatsapp-cli-mac
GOOS=windows GOARCH=amd64 go build -o whatsapp-cli.exe
```

## Data Access Patterns

### Read Operations (Direct SQLite)
- List messages: Query the `messages` table with filters.
- Search contacts: Query the `chats` table.
- Get chat metadata: Join the `chats` and `messages` tables.

### Write Operations (Via whatsmeow)
- Send a message: Use `client.SendMessage()`.
- Upload media: Use `client.Upload()`, then send it.
- Download media: Use `client.Download()`.

### Event Handling
- Real-time message sync: Event handlers process incoming messages.
- History sync: Handle `HistorySync` events for backfill.
- Connection status: Monitor connect and disconnect events.

## Comparison with Existing MCP Server

| Aspect | MCP Server | CLI Tool |
|--------|------------|----------|
| Architecture | Go bridge + Python MCP | Single Go binary |
| Dependencies | Go, Python, UV, Node (Claude) | Go only |
| Access Method | MCP protocol via stdio | Direct CLI invocation |
| Output Format | MCP tool results | JSON stdout |
| Authentication | QR code on bridge start | QR code on first command |
| Session Storage | SQLite (shared) | SQLite (same format) |
| Use Case | Claude Desktop integration | Codex, scripts, automation |

## Future Enhancements

- **Daemon mode**: Optional persistent connection for faster commands
- **Webhook support**: HTTP callbacks for incoming messages
- **Export**: Export chats to JSON, CSV, or HTML
- **Group management**: Create groups, manage participants
- **Status updates**: Post and view WhatsApp status
- **Typing indicators**: Send typing status
- **Read receipts**: Mark messages as read or unread
