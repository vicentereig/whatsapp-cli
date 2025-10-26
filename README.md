# WhatsApp CLI

A standalone command-line interface for WhatsApp, built on top of [whatsmeow](https://github.com/tulir/whatsmeow). Designed for automation, scripts, and use with AI coding assistants like Codex and Claude Code.

## Features

- **Standalone binary**: Single Go binary, no dependencies
- **Direct WhatsApp access**: Uses WhatsApp Web multidevice protocol
- **JSON output**: All commands return JSON for easy parsing
- **Persistent sessions**: QR code auth once, then automatic reconnection
- **Local storage**: All data stored locally in SQLite
- **Full messaging**: Read, send, search messages and contacts

## Installation

### Prerequisites

- Go 1.24+ (for building from source)

### Build from Source

```bash
git clone <repo>
cd whatsapp-cli
go build -o whatsapp-cli .
```

### Install

```bash
# Copy to PATH
sudo cp whatsapp-cli /usr/local/bin/

# Or use directly
./whatsapp-cli
```

## Quick Start

### 1. Authenticate

First time setup - scan QR code with your WhatsApp mobile app:

```bash
./whatsapp-cli auth
```

Output:
```
Scan this QR code with your WhatsApp app:
[QR CODE APPEARS]

✓ Successfully authenticated!
{"success":true,"data":{"authenticated":true,"message":"Successfully authenticated"},"error":null}
```

Session is saved to `./store/whatsapp.db` and persists for ~20 days.

### 2. List Chats

```bash
./whatsapp-cli chats list --limit 10
```

Output:
```json
{
  "success": true,
  "data": [
    {
      "jid": "1234567890@s.whatsapp.net",
      "name": "John Doe",
      "last_message_time": "2025-10-26T10:30:00Z"
    }
  ],
  "error": null
}
```

### 3. Search Contacts

```bash
./whatsapp-cli contacts search --query "John"
```

Output:
```json
{
  "success": true,
  "data": [
    {
      "phone_number": "1234567890",
      "name": "John Doe",
      "jid": "1234567890@s.whatsapp.net"
    }
  ],
  "error": null
}
```

### 4. List Messages

```bash
# List messages from a specific chat
./whatsapp-cli messages list --chat 1234567890@s.whatsapp.net --limit 20

# Search all messages
./whatsapp-cli messages search --query "meeting" --limit 50
```

Output:
```json
{
  "success": true,
  "data": [
    {
      "id": "msg123",
      "chat_jid": "1234567890@s.whatsapp.net",
      "chat_name": "John Doe",
      "sender": "1234567890",
      "content": "See you at the meeting!",
      "timestamp": "2025-10-26T10:30:00Z",
      "is_from_me": false
    }
  ],
  "error": null
}
```

### 5. Send Message

```bash
# Send to individual (phone number)
./whatsapp-cli send --to 1234567890 --message "Hello from CLI!"

# Send to group (use JID)
./whatsapp-cli send --to 123456789@g.us --message "Hello group!"
```

Output:
```json
{
  "success": true,
  "data": {
    "sent": true,
    "recipient": "1234567890",
    "message": "Hello from CLI!"
  },
  "error": null
}
```

## Commands Reference

### Global Options

- `--store DIR` - Storage directory (default: `./store`)

### `auth`

Authenticate with WhatsApp by scanning QR code.

```bash
whatsapp-cli auth
```

### `messages list`

List messages from all chats or a specific chat.

**Options:**
- `--chat JID` - Filter by chat JID
- `--limit N` - Limit results (default: 20)
- `--page N` - Page number (default: 0)

```bash
whatsapp-cli messages list --chat 1234567890@s.whatsapp.net --limit 50
```

### `messages search`

Search messages by content.

**Options:**
- `--query TEXT` - Search query
- `--limit N` - Limit results (default: 20)
- `--page N` - Page number (default: 0)

```bash
whatsapp-cli messages search --query "project update"
```

### `contacts search`

Search contacts by name or phone number.

**Options:**
- `--query TEXT` - Search query (required)

```bash
whatsapp-cli contacts search --query "John"
```

### `chats list`

List all chats sorted by most recent activity.

**Options:**
- `--query TEXT` - Filter chats by name/JID
- `--limit N` - Limit results (default: 20)
- `--page N` - Page number (default: 0)

```bash
whatsapp-cli chats list --limit 10
```

### `send`

Send a text message to a recipient.

**Options:**
- `--to RECIPIENT` - Phone number (e.g., `1234567890`) or JID (e.g., `1234567890@s.whatsapp.net` or `123@g.us` for groups)
- `--message TEXT` - Message text (required)

```bash
whatsapp-cli send --to 1234567890 --message "Hello!"
```

## Storage

All data is stored locally in the `--store` directory (default: `./store`):

```
store/
├── whatsapp.db      # Session data (managed by whatsmeow)
└── messages.db      # Message history
```

- **Session persists**: No need to re-authenticate unless session expires (~20 days)
- **Message history**: All received messages are stored automatically
- **Media metadata**: Media info stored, actual files downloaded on demand

## Usage with AI/Automation

### Parse JSON Output

All commands return JSON that's easy to parse:

```bash
# Get contact JID
JID=$(./whatsapp-cli contacts search --query "John" | jq -r '.data[0].jid')

# Send message to that contact
./whatsapp-cli send --to "$JID" --message "Hi John!"
```

### Example: Send daily reminder

```bash
#!/bin/bash
./whatsapp-cli send --to 1234567890 --message "Daily reminder: $(date)"
```

### Example with Claude Code

```typescript
// In Claude Code/Codex
const exec = require('child_process').execSync;

// Search contacts
const contacts = JSON.parse(
  exec('./whatsapp-cli contacts search --query "Team"').toString()
);

// Send to first result
if (contacts.success && contacts.data.length > 0) {
  const jid = contacts.data[0].jid;
  exec(`./whatsapp-cli send --to "${jid}" --message "Meeting in 10 mins"`);
}
```

## JID Format

WhatsApp uses JIDs (Jabber IDs) to identify chats:

- **Individual chats**: `phone_number@s.whatsapp.net` (e.g., `1234567890@s.whatsapp.net`)
- **Group chats**: `group_id@g.us` (e.g., `123456789@g.us`)

You can use either:
- Phone number alone (automatically converts to JID for individuals)
- Full JID (required for groups)

## Architecture

See [docs/architecture.md](docs/architecture.md) for detailed technical architecture.

**Key components:**
- `whatsmeow` - WhatsApp Web protocol implementation
- SQLite - Local storage for sessions and messages
- JSON output - Standard format for all responses

## Comparison with MCP Server

| Feature | CLI Tool | MCP Server |
|---------|----------|------------|
| Installation | Single binary | Go + Python + UV |
| Dependencies | None (compiled) | Multiple runtimes |
| Usage | Direct CLI invocation | MCP protocol via Claude Desktop |
| Output | JSON stdout | MCP tool results |
| Best for | Scripts, automation, AI agents | Claude Desktop integration |

## Troubleshooting

### "Not authenticated"

Run `./whatsapp-cli auth` to scan QR code.

### "Failed to connect"

- Check internet connection
- Ensure WhatsApp app is active on phone
- Session may have expired - re-authenticate

### "Database locked"

Only one instance can access the database at a time. Close other instances.

## Development

### Run Tests

```bash
go test ./...
```

### Build

```bash
go build -o whatsapp-cli .
```

### Cross-compile

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o whatsapp-cli-linux .

# macOS (ARM)
GOOS=darwin GOARCH=arm64 go build -o whatsapp-cli-mac .

# Windows
GOOS=windows GOARCH=amd64 go build -o whatsapp-cli.exe .
```

## License

[License info]

## Credits

Built with [whatsmeow](https://github.com/tulir/whatsmeow) by Tulir Asokan.
