# WhatsApp CLI - Complete Reference

> **For Humans & LLMs**: This document contains comprehensive information about the WhatsApp CLI tool, including installation, usage, API reference, examples, architecture, and troubleshooting. It is designed to be parsed by both humans and large language models.

**Version**: 1.3.2
**Repository**: https://github.com/vicentereig/whatsapp-cli
**License**: MIT
**Language**: Go 1.24+

---

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Complete Command Reference](#complete-command-reference)
- [JSON Response Format](#json-response-format)
- [Usage Examples](#usage-examples)
- [Integration with LLMs/AI Tools](#integration-with-llmsai-tools)
- [Data Model](#data-model)
- [Storage & Persistence](#storage--persistence)
- [Authentication & Security](#authentication--security)
- [Architecture](#architecture)
- [Error Handling](#error-handling)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
- [FAQ](#faq)
- [Credits](#credits)

---

## Overview

### What is WhatsApp CLI?

A standalone command-line interface for WhatsApp built on the WhatsApp Web multidevice protocol. Every command returns structured JSON output, making it ideal for:

- **Automation**: Shell scripts, cron jobs, CI/CD pipelines
- **AI Integration**: Codex, Claude Code, GPT-based tools
- **Data Analysis**: Extract and analyze WhatsApp conversations
- **Custom Applications**: Build tools on top of WhatsApp

### Key Features

| Feature | Description |
|---------|-------------|
| **Zero Dependencies** | Single compiled binary (21MB), no runtime required |
| **JSON Output** | All commands return structured JSON for easy parsing |
| **Persistent Sessions** | Authenticate once via QR code, auto-reconnect for ~20 days |
| **Local Storage** | SQLite database, no cloud dependencies |
| **Full Messaging** | Send, receive, search messages; manage contacts & chats |
| **Group Support** | Send/receive messages in group chats |
| **TDD Implementation** | 100% test coverage, production-ready |

### System Requirements

- **Operating System**: Linux, macOS, Windows
- **Architecture**: x86_64, ARM64
- **Go Version**: 1.24+ (for building from source)
- **Storage**: ~50MB for binary + variable for message database
- **Network**: Internet connection required
- **WhatsApp Account**: Active WhatsApp account with smartphone

---

## Installation

### Method 1: Homebrew (Recommended)

```bash
brew install vicentereig/tap/whatsapp-cli
```

Or tap first, then install:

```bash
brew tap vicentereig/tap
brew install whatsapp-cli
```

### Method 2: Download Pre-built Binary

```bash
# Linux (x86_64)
curl -LO https://github.com/vicentereig/whatsapp-cli/releases/latest/download/whatsapp-cli-linux-amd64.tar.gz
tar -xzf whatsapp-cli-linux-amd64.tar.gz
sudo mv whatsapp-cli-linux-amd64 /usr/local/bin/whatsapp-cli

# macOS (ARM64 - M1/M2/M3)
curl -LO https://github.com/vicentereig/whatsapp-cli/releases/latest/download/whatsapp-cli-darwin-arm64.tar.gz
tar -xzf whatsapp-cli-darwin-arm64.tar.gz
sudo mv whatsapp-cli-darwin-arm64 /usr/local/bin/whatsapp-cli

# macOS (Intel)
curl -LO https://github.com/vicentereig/whatsapp-cli/releases/latest/download/whatsapp-cli-darwin-amd64.tar.gz
tar -xzf whatsapp-cli-darwin-amd64.tar.gz
sudo mv whatsapp-cli-darwin-amd64 /usr/local/bin/whatsapp-cli

# Windows (x86_64) - download and extract whatsapp-cli-windows-amd64.zip from releases page
```

### Method 3: Build from Source

```bash
# Clone repository
git clone https://github.com/vicentereig/whatsapp-cli.git
cd whatsapp-cli

# Install dependencies
go mod download

# Build
go build -o whatsapp-cli .

# Install (optional)
sudo mv whatsapp-cli /usr/local/bin/

# Verify installation
whatsapp-cli --help
```

### Method 4: Install via Go

```bash
go install github.com/vicentereig/whatsapp-cli@latest
```

### Release & Distribution Notes

- Version tags use semantic versioning (`vMAJOR.MINOR.PATCH`). Use a specific tag (e.g., `v1.0.0`) with `go install github.com/vicentereig/whatsapp-cli@v1.0.0` for reproducible builds.
- Pre-built artifacts for Linux/macOS/Windows are published on the [GitHub Releases](https://github.com/vicentereig/whatsapp-cli/releases) page. Each archive is paired with SHA-256 entries inside `checksums.txt`; run `shasum -a 256 -c checksums.txt --ignore-missing` to verify before installing.
- Binaries are named `whatsapp-cli-<os>-<arch>` (Windows adds `.exe`). After extraction, mark them executable (`chmod +x`) and place them somewhere in your `PATH` such as `/usr/local/bin/whatsapp-cli`.
- Each uploaded file (including `checksums.txt`) also has Sigstore cosign signatures (`.sig`) and certificates (`.pem`). To verify, run `cosign verify-blob --certificate <file>.pem --signature <file>.sig <file>`; GitHub Actions uses OIDC identities, so you can enforce provenance on verification.
- To build from source, follow Method 3. That path is ideal when you want to audit the code, tweak compilation flags, or test changes before tagging a release.
- Maintainers can follow `docs/RELEASE.md` for the step-by-step process that drives `go install`, source builds, and GitHub Release automation.

---

## Quick Start

### Step 1: First-Time Authentication

```bash
whatsapp-cli auth
```

**What happens:**
1. QR code appears in terminal
2. Open WhatsApp on your phone → Settings → Linked Devices → Link a Device
3. Scan the QR code
4. Session saved to `./store/whatsapp.db`

**Output:**
```json
{
  "success": true,
  "data": {
    "authenticated": true,
    "message": "Successfully authenticated"
  },
  "error": null
}
```

**Session Duration**: ~20 days before re-authentication required

### Step 2: Sync Messages

Before you can list or search messages, you need to sync them from WhatsApp:

```bash
# Start syncing messages (run this in the background or a separate terminal)
whatsapp-cli sync
# Press Ctrl+C when done syncing
```

**What happens:**
1. Connects to WhatsApp and stays connected
2. Downloads message history from WhatsApp servers
3. Receives new messages in real-time
4. Stores everything in `./store/messages.db`
5. Runs until you press Ctrl+C

**Tip**: Run sync in a tmux/screen session or as a background service to continuously receive messages.

### Step 3: Basic Operations

```bash
# List your chats
whatsapp-cli chats list --limit 10

# Search for a contact
whatsapp-cli contacts search --query "John"

# Send a message
whatsapp-cli send --to 1234567890 --message "Hello from CLI!"

# Search messages
whatsapp-cli messages search --query "meeting"
```

### Step 4: Check the Installed Version

```bash
whatsapp-cli version
```

**Example Output:**

```json
{
  "success": true,
  "data": {
    "version": "v1.1.0"
  },
  "error": null
}
```

---

## Complete Command Reference

### Global Options

All commands support these global flags:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--store` | string | `./store` | Directory for session and message databases |

**Example:**
```bash
whatsapp-cli --store /var/lib/whatsapp chats list
```

---

### Command: `auth`

Authenticate with WhatsApp via QR code.

**Syntax:**
```bash
whatsapp-cli auth
```

**Parameters:** None

**Returns:**
```json
{
  "success": true,
  "data": {
    "authenticated": boolean,
    "message": string
  },
  "error": null
}
```

**Behavior:**
- If already authenticated: Returns success immediately
- If not authenticated: Displays QR code and waits for scan
- Timeout: 5 minutes
- Creates `store/whatsapp.db` with session data

**Example:**
```bash
whatsapp-cli auth
# Scan QR code with phone
# ✓ Successfully authenticated!
```

---

### Command: `sync`

**⚠️ IMPORTANT**: This command must be run to populate the message database. Without running sync, `messages list` and `messages search` will return empty results.

Continuously sync messages from WhatsApp to local database. This command:
- Downloads message history from WhatsApp servers
- Receives new incoming messages in real-time
- Stores all messages in SQLite database
- Runs until you press Ctrl+C

**Syntax:**
```bash
whatsapp-cli sync
```

**Parameters:** None

**Returns:** (on exit via Ctrl+C)
```json
{
  "success": true,
  "data": {
    "synced": true,
    "messages_count": 1234
  },
  "error": null
}
```

**Behavior:**
- Connects to WhatsApp (authenticates if needed)
- Registers event handlers for incoming messages and history sync
- Processes `*events.Message` for real-time messages
- Processes `*events.HistorySync` for message history batches
- Stores all messages in `store/messages.db`
- Updates progress to stderr (doesn't interfere with JSON output)
- Runs indefinitely until interrupted (Ctrl+C)
- Gracefully disconnects on exit

**Progress Output (stderr):**
```
🚀 Starting WhatsApp sync...
✓ Connected to WhatsApp
🔄 Listening for messages... (Press Ctrl+C to stop)
📜 Processing history sync (42 conversations)...
💬 Synced 1234 messages...
^C
✓ Sync completed. Total messages synced: 1234
```

**Examples:**

```bash
# Basic sync - run in foreground
whatsapp-cli sync

# Run in background (recommended for continuous syncing)
whatsapp-cli sync > sync.json 2> sync.log &

# Run in tmux/screen session
tmux new -s whatsapp
whatsapp-cli sync
# Detach with Ctrl+B D

# Run with custom storage directory
whatsapp-cli --store /var/lib/whatsapp sync

# Stop sync gracefully
kill -INT <pid>
# Or press Ctrl+C in foreground
```

**Use Cases:**
1. **Initial Setup**: Run once to download all message history
2. **Continuous Sync**: Run as background service to receive messages
3. **Periodic Sync**: Run via cron to update messages periodically
4. **Development**: Run in terminal while testing queries

**Notes:**
- Message history sync may take time depending on message count
- Duplicate messages are handled by SQLite PRIMARY KEY constraints
- Media files are NOT downloaded, only metadata (type, filename, URL)
- Connection stays alive indefinitely until interrupted
- Safe to restart - won't duplicate messages

---

### Command: `messages list`

List messages from all chats or a specific chat.

**Syntax:**
```bash
whatsapp-cli messages list [OPTIONS]
```

**Parameters:**

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--chat` | string | No | - | Filter by chat JID (e.g., `1234567890@s.whatsapp.net`) |
| `--limit` | int | No | 20 | Maximum number of messages to return |
| `--page` | int | No | 0 | Page number for pagination (0-indexed) |

**Returns:**
```json
{
  "success": true,
  "data": [
    {
      "id": "msg_unique_id",
      "chat_jid": "1234567890@s.whatsapp.net",
      "chat_name": "John Doe",
      "sender": "1234567890",
      "content": "Message text content",
      "timestamp": "2025-10-26T10:30:00Z",
      "is_from_me": false,
      "media_type": ""
    }
  ],
  "error": null
}
```

**Examples:**
```bash
# List 50 most recent messages across all chats
whatsapp-cli messages list --limit 50

# List messages from specific chat
whatsapp-cli messages list --chat 1234567890@s.whatsapp.net

# Pagination: Get second page of results
whatsapp-cli messages list --limit 20 --page 1

# Get JID first, then list messages
JID=$(whatsapp-cli contacts search --query "Alice" | jq -r '.data[0].jid')
whatsapp-cli messages list --chat "$JID" --limit 100
```

**Sorting:** Messages returned in reverse chronological order (newest first)

---

### Command: `messages search`

Search messages by content across all chats.

**Syntax:**
```bash
whatsapp-cli messages search --query TEXT [OPTIONS]
```

**Parameters:**

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--query` | string | Yes | - | Search term (case-insensitive, partial match) |
| `--limit` | int | No | 20 | Maximum number of results |
| `--page` | int | No | 0 | Page number for pagination |

**Returns:** Same format as `messages list`

**Examples:**
```bash
# Search for messages containing "meeting"
whatsapp-cli messages search --query "meeting"

# Search with more results
whatsapp-cli messages search --query "project" --limit 100

# Case-insensitive search
whatsapp-cli messages search --query "URGENT"  # Finds "urgent", "Urgent", etc.
```

**Search Behavior:**
- Case-insensitive
- Partial word matching
- Searches message content only (not sender names)
- Returns messages from all chats

---

### Command: `contacts search`

Search contacts by name or phone number.

**Syntax:**
```bash
whatsapp-cli contacts search --query TEXT
```

**Parameters:**

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--query` | string | Yes | - | Search term for name or phone number |

**Returns:**
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

**Examples:**
```bash
# Search by name
whatsapp-cli contacts search --query "John"

# Search by partial phone number
whatsapp-cli contacts search --query "5551234"

# Extract JID for further operations
JID=$(whatsapp-cli contacts search --query "Alice" | jq -r '.data[0].jid')
echo "Alice's JID: $JID"
```

**Behavior:**
- Returns maximum 50 results
- Excludes group chats (only individual contacts)
- Sorted alphabetically by name
- Partial matching on both name and JID

---

### Command: `chats list`

List all chats sorted by recent activity.

**Syntax:**
```bash
whatsapp-cli chats list [OPTIONS]
```

**Parameters:**

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--query` | string | No | - | Filter chats by name or JID |
| `--limit` | int | No | 20 | Maximum number of chats |
| `--page` | int | No | 0 | Page number for pagination |

**Returns:**
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

**Examples:**
```bash
# List 20 most recent chats
whatsapp-cli chats list

# List all chats (with pagination)
whatsapp-cli chats list --limit 100

# Filter chats by name
whatsapp-cli chats list --query "Team"

# Get list of all group chats
whatsapp-cli chats list | jq '.data[] | select(.jid | endswith("@g.us"))'
```

**Sorting:** Chats ordered by `last_message_time` (most recent first)

**Chat Types:**
- Individual chats: JID ends with `@s.whatsapp.net`
- Group chats: JID ends with `@g.us`

---

### Command: `send`

Send a text message to an individual or group.

**Syntax:**
```bash
whatsapp-cli send --to RECIPIENT --message TEXT
```

**Parameters:**

| Flag | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `--to` | string | Yes | - | Phone number or JID |
| `--message` | string | Yes | - | Message text content |

**Recipient Formats:**

| Format | Example | Use Case |
|--------|---------|----------|
| Phone number | `1234567890` | Individual chats (auto-converted to JID) |
| Individual JID | `1234567890@s.whatsapp.net` | Individual chats |
| Group JID | `123456789@g.us` | Group chats (must use JID) |

**Returns:**
```json
{
  "success": true,
  "data": {
    "sent": true,
    "recipient": "1234567890",
    "message": "Hello!"
  },
  "error": null
}
```

**Examples:**
```bash
# Send to individual (phone number)
whatsapp-cli send --to 1234567890 --message "Hello from CLI!"

# Send to individual (full JID)
whatsapp-cli send --to 1234567890@s.whatsapp.net --message "Hi there!"

# Send to group (requires JID)
whatsapp-cli send --to 123456789@g.us --message "Hello everyone!"

# Send with special characters (use quotes)
whatsapp-cli send --to 1234567890 --message "It's working! 🎉"

# Multi-line messages
whatsapp-cli send --to 1234567890 --message "Line 1
Line 2
Line 3"

# Send result of command
whatsapp-cli send --to 1234567890 --message "Server status: $(uptime)"
```

**Behavior:**
- Requires active connection (authenticates if needed)
- Message stored locally in database
- Returns immediately after sending (does not wait for delivery)
- Supports Unicode (emojis, international characters)

**Limitations:**
- Send command currently supports text only (download attachments via `media download`)
- No delivery/read receipt information returned
- Maximum message length: WhatsApp's standard limit (~65,536 characters)

---

### Command: `media download`

Download media attachments (images, videos, audio, documents) that were synced into the local database.

**Syntax:**
```bash
whatsapp-cli media download --message-id ID [--chat JID] [--output PATH]
```

**Parameters:**

| Flag | Type | Required | Description |
|------|------|----------|-------------|
| `--message-id` | string | Yes | Message identifier from `messages list/search` |
| `--chat` | string | No | Chat JID to disambiguate duplicate message IDs |
| `--output` | string | No | Destination file or directory (defaults to auto-structured path) |

**Default storage:**
- Media is stored next to the SQLite databases under `STORE/media/{chat}/{message}/{media_type}/filename`
- Paths are sanitized automatically
- If `--output` points to a directory, the original filename (or message ID-based fallback) is used

**Return value:**
```json
{
  "success": true,
  "data": {
    "message_id": "ABCD1234",
    "chat_jid": "1234567890@s.whatsapp.net",
    "path": "/path/to/media/1234567890@s.whatsapp.net/ABCD1234/image/ABCD1234.jpg",
    "bytes": 204800,
    "media_type": "image",
    "mime_type": "image/jpeg",
    "downloaded_at": "2025-02-01T12:34:56.789Z"
  },
  "error": null
}
```

**Examples:**
```bash
# Download image using auto-organised directory layout
whatsapp-cli media download --message-id ABCD1234

# Save into a specific directory (filename auto-generated)
whatsapp-cli media download --message-id ABCD1234 --output /tmp/media/

# Save using explicit path and disambiguate by chat JID
whatsapp-cli media download --message-id XYZ987 --chat 1234567890@s.whatsapp.net --output ~/Downloads/report.pdf
```

**Notes:**
- Requires that `whatsapp-cli sync` has captured the message metadata
- The sync loop downloads media concurrently in the background without blocking new messages
- Re-running the command overwrites the existing file with a fresh download
- Errors include metadata issues (expired link, missing direct path) or filesystem permissions

---

## JSON Response Format

All commands return JSON in this standardized format:

### Success Response

```json
{
  "success": true,
  "data": <result_data>,
  "error": null
}
```

### Error Response

```json
{
  "success": false,
  "data": null,
  "error": "Error message describing what went wrong"
}
```

### Data Types by Command

| Command | Data Type | Structure |
|---------|-----------|-----------|
| `auth` | object | `{"authenticated": bool, "message": string}` |
| `messages list` | array | `[Message, ...]` |
| `messages search` | array | `[Message, ...]` |
| `contacts search` | array | `[Contact, ...]` |
| `chats list` | array | `[Chat, ...]` |
| `send` | object | `{"sent": bool, "recipient": string, "message": string}` |

---

## Usage Examples

### Example 1: Send Daily Report via Cron

```bash
#!/bin/bash
# File: /usr/local/bin/daily-report.sh

RECIPIENT="1234567890"  # Your phone number

# Generate report
REPORT=$(cat <<EOF
📊 Daily Report - $(date +%Y-%m-%d)

Server Status: $(systemctl is-active nginx)
Disk Usage: $(df -h / | awk 'NR==2 {print $5}')
Memory: $(free -h | awk 'NR==2 {print $3 "/" $2}')
Uptime: $(uptime -p)
EOF
)

# Send via WhatsApp
whatsapp-cli send --to "$RECIPIENT" --message "$REPORT"
```

**Cron entry:**
```cron
0 9 * * * /usr/local/bin/daily-report.sh
```

### Example 2: Search and Bulk Message

```bash
#!/bin/bash
# Send message to all contacts matching "Team"

CONTACTS=$(whatsapp-cli contacts search --query "Team" | jq -r '.data[].jid')

for JID in $CONTACTS; do
  whatsapp-cli send --to "$JID" --message "Team meeting at 3 PM today!"
  sleep 2  # Rate limiting
done
```

### Example 3: Export Chat History

```bash
#!/bin/bash
# Export specific chat to JSON file

JID="1234567890@s.whatsapp.net"
OUTPUT_FILE="chat_export_$(date +%Y%m%d).json"

# Get all messages (paginated)
PAGE=0
ALL_MESSAGES=[]

while true; do
  RESPONSE=$(whatsapp-cli messages list --chat "$JID" --limit 100 --page $PAGE)
  MESSAGES=$(echo "$RESPONSE" | jq '.data')
  COUNT=$(echo "$MESSAGES" | jq 'length')

  if [ "$COUNT" -eq 0 ]; then
    break
  fi

  ALL_MESSAGES=$(echo "$ALL_MESSAGES" | jq ". + $MESSAGES")
  PAGE=$((PAGE + 1))
done

echo "$ALL_MESSAGES" | jq '.' > "$OUTPUT_FILE"
echo "Exported to $OUTPUT_FILE"
```

### Example 4: Monitor for Keywords

```bash
#!/bin/bash
# Alert when specific keywords appear in messages

ALERT_RECIPIENT="your_phone@s.whatsapp.net"
LAST_CHECK_FILE="/tmp/whatsapp_last_check"

# Get timestamp of last check
if [ -f "$LAST_CHECK_FILE" ]; then
  LAST_CHECK=$(cat "$LAST_CHECK_FILE")
else
  LAST_CHECK=$(date -u -d "1 hour ago" +%Y-%m-%dT%H:%M:%SZ)
fi

# Search for urgent messages since last check
MESSAGES=$(whatsapp-cli messages search --query "URGENT" --limit 100 | \
  jq --arg since "$LAST_CHECK" '.data[] | select(.timestamp > $since)')

if [ -n "$MESSAGES" ]; then
  COUNT=$(echo "$MESSAGES" | jq -s 'length')
  whatsapp-cli send --to "$ALERT_RECIPIENT" \
    --message "⚠️ $COUNT urgent messages found!"
fi

# Update last check timestamp
date -u +%Y-%m-%dT%H:%M:%SZ > "$LAST_CHECK_FILE"
```

---

## Integration with LLMs/AI Tools

### Parsing JSON with `jq`

```bash
# Extract specific fields
whatsapp-cli contacts search --query "John" | jq '.data[0].jid'
# Output: "1234567890@s.whatsapp.net"

# Count results
whatsapp-cli chats list | jq '.data | length'
# Output: 42

# Filter and transform
whatsapp-cli messages list | jq '[.data[] | {name: .chat_name, msg: .content}]'
```

### Python Integration

```python
#!/usr/bin/env python3
import subprocess
import json

def whatsapp_cli(command):
    """Execute whatsapp-cli command and return parsed JSON."""
    result = subprocess.run(
        ['whatsapp-cli'] + command.split(),
        capture_output=True,
        text=True
    )
    return json.loads(result.stdout)

# Search contacts
contacts = whatsapp_cli('contacts search --query "Team"')
if contacts['success']:
    for contact in contacts['data']:
        print(f"{contact['name']}: {contact['jid']}")

# Send message
result = whatsapp_cli('send --to 1234567890 --message "Hello from Python!"')
print(f"Message sent: {result['success']}")
```

### Node.js/TypeScript Integration

```typescript
import { execSync } from 'child_process';

interface WhatsAppResponse<T> {
  success: boolean;
  data: T | null;
  error: string | null;
}

function whatsappCli<T>(command: string): WhatsAppResponse<T> {
  const output = execSync(`whatsapp-cli ${command}`).toString();
  return JSON.parse(output);
}

// Usage
const contacts = whatsappCli<Contact[]>('contacts search --query "Alice"');
if (contacts.success && contacts.data) {
  const jid = contacts.data[0].jid;
  whatsappCli(`send --to ${jid} --message "Hi from Node!"`);
}
```

### Claude Code / Codex Integration

```typescript
// Example Claude Code MCP tool wrapper

import { execSync } from 'child_process';

export const whatsappTools = {
  sendMessage: async (to: string, message: string) => {
    const result = execSync(
      `whatsapp-cli send --to "${to}" --message "${message}"`
    ).toString();
    return JSON.parse(result);
  },

  searchContacts: async (query: string) => {
    const result = execSync(
      `whatsapp-cli contacts search --query "${query}"`
    ).toString();
    return JSON.parse(result);
  },

  getRecentMessages: async (chatJid: string, limit = 20) => {
    const result = execSync(
      `whatsapp-cli messages list --chat "${chatJid}" --limit ${limit}`
    ).toString();
    return JSON.parse(result);
  }
};

// Use in Claude Code
const contacts = await whatsappTools.searchContacts("Alice");
if (contacts.success) {
  await whatsappTools.sendMessage(
    contacts.data[0].jid,
    "Automated message from Claude Code!"
  );
}
```

---

## Data Model

### Message Object

```typescript
interface Message {
  id: string;                    // Unique message ID
  chat_jid: string;              // Chat identifier (JID)
  chat_name: string;             // Display name of chat
  sender: string;                // Phone number of sender
  content: string;               // Message text content
  timestamp: string;             // ISO 8601 timestamp
  is_from_me: boolean;           // true if sent by you
  media_type?: string;           // "image", "video", "audio", "document", or ""
}
```

**Example:**
```json
{
  "id": "3EB0F2A8B9C4D1E5F6A7",
  "chat_jid": "1234567890@s.whatsapp.net",
  "chat_name": "John Doe",
  "sender": "1234567890",
  "content": "See you at the meeting!",
  "timestamp": "2025-10-26T14:30:00Z",
  "is_from_me": false,
  "media_type": ""
}
```

### Contact Object

```typescript
interface Contact {
  phone_number: string;          // Phone number (without country code prefix)
  name: string;                  // Display name
  jid: string;                   // WhatsApp JID
}
```

**Example:**
```json
{
  "phone_number": "1234567890",
  "name": "John Doe",
  "jid": "1234567890@s.whatsapp.net"
}
```

### Chat Object

```typescript
interface Chat {
  jid: string;                   // Chat identifier
  name: string;                  // Display name
  last_message_time: string;     // ISO 8601 timestamp of last message
}
```

**Example:**
```json
{
  "jid": "123456789@g.us",
  "name": "Project Team",
  "last_message_time": "2025-10-26T16:45:00Z"
}
```

### JID (Jabber ID) Format

WhatsApp uses JIDs to uniquely identify chats:

| Type | Format | Example |
|------|--------|---------|
| Individual | `{phone}@s.whatsapp.net` | `1234567890@s.whatsapp.net` |
| Group | `{group_id}@g.us` | `120363012345678901@g.us` |

**Extracting Components:**
```bash
# Get phone number from individual JID
echo "1234567890@s.whatsapp.net" | cut -d'@' -f1
# Output: 1234567890

# Check if JID is a group
echo "$JID" | grep -q "@g.us" && echo "Group" || echo "Individual"
```

---

## Storage & Persistence

### Database Location

Default storage directory: `./store/`

```
store/
├── whatsapp.db      # Session data (managed by whatsmeow)
└── messages.db      # Message history (managed by CLI)
```

**Custom Location:**
```bash
whatsapp-cli --store /var/lib/whatsapp chats list
```

### Session Database (`whatsapp.db`)

- **Format**: SQLite3
- **Managed by**: whatsmeow library
- **Contents**:
  - Device ID and encryption keys
  - Contact information
  - Session tokens
- **Persistence**: ~20 days before re-authentication required
- **Security**: Contains sensitive data, protect with file permissions

**Recommended Permissions:**
```bash
chmod 600 store/whatsapp.db
chown $USER:$USER store/whatsapp.db
```

### Message Database (`messages.db`)

- **Format**: SQLite3
- **Managed by**: WhatsApp CLI
- **Schema**:

```sql
-- Chats table
CREATE TABLE chats (
    jid TEXT PRIMARY KEY,
    name TEXT,
    last_message_time TIMESTAMP
);

-- Messages table
CREATE TABLE messages (
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
```

### Direct Database Access

```bash
# Open with sqlite3
sqlite3 store/messages.db

# Query examples
sqlite> SELECT COUNT(*) FROM messages;
sqlite> SELECT chat_name, COUNT(*) FROM messages
        JOIN chats ON messages.chat_jid = chats.jid
        GROUP BY chat_name;
sqlite> .quit
```

### Backup & Migration

```bash
# Backup
tar -czf whatsapp-backup-$(date +%Y%m%d).tar.gz store/

# Restore
tar -xzf whatsapp-backup-20251026.tar.gz

# Migrate to new machine
rsync -avz store/ newserver:/path/to/store/
```

---

## Authentication & Security

### Authentication Flow

1. **First Run**: QR code displayed
2. **User Action**: Scan with WhatsApp mobile app
3. **Session Creation**: Credentials saved to `store/whatsapp.db`
4. **Subsequent Runs**: Auto-reconnect using saved session
5. **Expiration**: After ~20 days, re-authentication required

### Security Considerations

#### Session Protection

**The `store/whatsapp.db` file contains your WhatsApp session credentials.** Treat it like a password:

```bash
# Set proper permissions
chmod 600 store/whatsapp.db
chmod 700 store/

# Never commit to version control
echo "store/" >> .gitignore

# Use environment variable for custom location
export WHATSAPP_STORE="/secure/path/to/store"
whatsapp-cli --store "$WHATSAPP_STORE" chats list
```

#### Multi-Device Limit

WhatsApp allows up to 5 linked devices. If you reach this limit:

1. Open WhatsApp on phone
2. Go to Settings → Linked Devices
3. Remove an old device
4. Re-authenticate CLI

#### Network Security

- All communication with WhatsApp uses end-to-end encryption
- CLI communicates via WhatsApp Web protocol (WSS)
- No data sent to third parties
- Local storage only

### Privacy

- **Message Storage**: All messages stored locally in SQLite
- **No Cloud Sync**: Data never leaves your machine
- **Media**: Metadata stored, actual files downloaded on-demand
- **Deletion**: Delete `store/` directory to remove all data

---

## Architecture

### System Design

```
┌─────────────────────────────────────────────────┐
│                  whatsapp-cli                    │
│                   (Go Binary)                    │
├─────────────────────────────────────────────────┤
│  ┌───────────┐  ┌──────────┐  ┌─────────────┐  │
│  │  Commands │  │  Client  │  │   Storage   │  │
│  │   Layer   │→ │  Wrapper │→ │   (SQLite)  │  │
│  └───────────┘  └──────────┘  └─────────────┘  │
│        ↓             ↓                           │
│  ┌───────────┐  ┌──────────┐                    │
│  │   Output  │  │whatsmeow │                    │
│  │   (JSON)  │  │ Library  │                    │
│  └───────────┘  └──────────┘                    │
│                      ↓                           │
└──────────────────────┼──────────────────────────┘
                       ↓
              ┌────────────────┐
              │  WhatsApp Web  │
              │   API (WSS)    │
              └────────────────┘
```

### Component Layers

| Layer | Location | Responsibility |
|-------|----------|----------------|
| **CLI** | `main.go` | Argument parsing, command routing |
| **Commands** | `internal/commands/` | Business logic for each command |
| **Client** | `internal/client/` | WhatsApp protocol wrapper |
| **Storage** | `internal/store/` | Database operations |
| **Output** | `internal/output/` | JSON formatting |

### Dependencies

```
whatsmeow (github.com/tulir/whatsmeow)
├── WebSocket communication
├── Protocol buffer encoding/decoding
├── End-to-end encryption
└── Session management

go-sqlite3 (github.com/mattn/go-sqlite3)
└── SQLite database driver

qrterminal (github.com/mdp/qrterminal)
└── QR code rendering in terminal
```

### Build Process

```bash
# Build with all dependencies statically linked
go build -ldflags="-s -w" -o whatsapp-cli .

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o whatsapp-cli-linux .
GOOS=darwin GOARCH=arm64 go build -o whatsapp-cli-mac .
GOOS=windows GOARCH=amd64 go build -o whatsapp-cli.exe .
```

**Binary Size Optimization:**
```bash
# Standard build
go build -o whatsapp-cli .          # ~21MB

# Optimized build
go build -ldflags="-s -w" .         # ~15MB

# Ultra-compressed (requires UPX)
upx --best --lzma whatsapp-cli      # ~5MB
```

For detailed technical architecture, see [docs/architecture.md](docs/architecture.md).

---

## Error Handling

### Common Error Responses

#### Not Authenticated
```json
{
  "success": false,
  "data": null,
  "error": "not authenticated"
}
```
**Solution:** Run `whatsapp-cli auth`

#### Connection Failed
```json
{
  "success": false,
  "data": null,
  "error": "failed to connect: connection refused"
}
```
**Solutions:**
- Check internet connection
- Verify WhatsApp is active on phone
- Check firewall rules

#### Invalid JID
```json
{
  "success": false,
  "data": null,
  "error": "invalid JID format"
}
```
**Solution:** Ensure JID format is correct (`phone@s.whatsapp.net` or `id@g.us`)

#### Database Locked
```json
{
  "success": false,
  "data": null,
  "error": "database is locked"
}
```
**Solution:** Close other instances of whatsapp-cli

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (check JSON error field) |

---

## Troubleshooting

### Authentication Issues

**Problem**: QR code doesn't appear

```bash
# Check terminal supports UTF-8
echo $LANG
# Should output: en_US.UTF-8 or similar

# Try with different QR code size
QRSIZE=small whatsapp-cli auth
```

**Problem**: "Session expired" after a few days

- WhatsApp sessions expire after ~20 days
- Re-run `whatsapp-cli auth`
- Sessions also expire if you unlink device from phone

### Connection Issues

**Problem**: "Connection timeout"

```bash
# Test network connectivity
ping -c 3 web.whatsapp.com

# Check DNS resolution
nslookup web.whatsapp.com

# Try with custom store path (permission issues)
whatsapp-cli --store /tmp/wa-test auth
```

**Problem**: "WebSocket connection failed"

- Corporate firewalls may block WSS
- VPN may interfere with connection
- Try from different network

### WhatsApp Web Version Bumps (405 / close 1006)

WhatsApp occasionally requires a newer desktop/web build. When that happens, the bundled `whatsmeow` dependency logs `Client outdated (405)` and immediately closes the websocket with code `1006`, so `whatsapp-cli` dies right after scanning the QR.

To fix it:

1. **Bump the dependency:**
   ```bash
   go get go.mau.fi/whatsmeow@latest
   go mod tidy
   ```
2. **Rebuild the binary:**
   ```bash
   go build -o whatsapp-cli .
   ```
3. **Verify:** Run `./whatsapp-cli version` followed by `./whatsapp-cli auth` or `./whatsapp-cli sync`. You should see `✓ Connected to WhatsApp` without any `Client outdated (405)` messages.

If the upstream library has not yet released a new version, monitor `tulir/whatsmeow` issues for `Client outdated (405)` reports; once a fix lands, repeat the steps above to update your local binary.

### Database Issues

**Problem**: "Database is locked"

```bash
# Check for other processes
ps aux | grep whatsapp-cli

# Check for stale lock files
rm -f store/*.db-shm store/*.db-wal

# If corrupted, restore from backup
mv store/messages.db store/messages.db.bak
# Re-run cli to recreate
```

**Problem**: "Disk full" or "No space left"

```bash
# Check database size
du -h store/messages.db

# Vacuum database to reclaim space
sqlite3 store/messages.db "VACUUM;"

# Delete old messages
sqlite3 store/messages.db "DELETE FROM messages WHERE timestamp < datetime('now', '-30 days');"
```

### Message Issues

**Problem**: Messages not appearing in searches

- CLI only searches messages stored locally
- Messages received before CLI was running are not stored
- Re-sync by listing messages from specific chats

**Problem**: Can't send to group

- Must use group JID (ending in `@g.us`), not phone number
- Get group JID from `chats list` command

### Performance Issues

**Problem**: Slow searches on large databases

```bash
# Add indexes
sqlite3 store/messages.db <<EOF
CREATE INDEX IF NOT EXISTS idx_content ON messages(content);
CREATE INDEX IF NOT EXISTS idx_timestamp ON messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_chat_jid ON messages(chat_jid);
EOF
```

### Debug Mode

```bash
# Enable verbose logging (if implemented)
export WHATSAPP_DEBUG=1
whatsapp-cli chats list

# Check SQLite directly
sqlite3 store/messages.db "SELECT * FROM messages LIMIT 5;"
```

---

## Development

### Setting Up Development Environment

```bash
# Clone repository
git clone https://github.com/vicentereig/whatsapp-cli.git
cd whatsapp-cli

# Install dependencies
go mod download

# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Build development version
go build -o whatsapp-cli-dev .
```

### Building with a Local Module Cache

If your environment can't write to the global Go module cache (common in sandboxes or CI), point `GOMODCACHE` to a writable directory within the repo before building:

```bash
GOMODCACHE=$(pwd)/.gomodcache go build ./...
```

This command compiles the full project and stores downloaded modules under `.gomodcache`, keeping the workspace self-contained.

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/store

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Verbose output
go test -v ./...

# Run specific test
go test -run TestStoreMessage ./internal/store
```

### Code Structure

```
internal/
├── client/
│   └── client.go          # WhatsApp client wrapper
├── commands/
│   └── commands.go        # CLI command implementations
├── output/
│   ├── output.go          # JSON formatting
│   └── output_test.go     # Output tests
└── store/
    ├── store.go           # Database operations
    └── store_test.go      # Storage tests
```

### Adding New Commands

1. **Define command in `internal/commands/commands.go`:**
```go
func (a *App) NewCommand(param string) string {
    // Implementation
    result := doSomething(param)
    return output.Success(result)
}
```

2. **Add routing in `main.go`:**
```go
case "newcommand":
    cmdFlags := flag.NewFlagSet("newcommand", flag.ExitOnError)
    param := cmdFlags.String("param", "", "description")
    cmdFlags.Parse(args[1:])
    result = app.NewCommand(*param)
```

3. **Write tests in `internal/commands/commands_test.go`:**
```go
func TestNewCommand(t *testing.T) {
    // Test implementation
}
```

### Testing Guide

Follow TDD (Test-Driven Development):

1. **Write test first**
2. **Run test (should fail)**
3. **Implement feature**
4. **Run test (should pass)**
5. **Refactor**

**Example:**
```go
// store_test.go
func TestStoreMessage(t *testing.T) {
    store := setupTestDB(t)
    err := store.StoreMessage("id1", "chat@s.whatsapp.net", "sender",
        "Hello", time.Now(), false, "", "", "", nil, nil, nil, 0)
    assert.NoError(t, err)
}
```

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## FAQ

### General

**Q: Is this an official WhatsApp tool?**
A: No, this is a third-party tool using the unofficial WhatsApp Web protocol via whatsmeow.

**Q: Can I use this with WhatsApp Business?**
A: Yes, works with both personal and business accounts.

**Q: Does this work on multiple devices simultaneously?**
A: Yes, WhatsApp supports up to 5 linked devices.

### Features

**Q: Can I send images/videos?**
A: Not yet. Currently text-only. Media support planned for future release.

**Q: Can I create/manage groups?**
A: Not yet. Read-only access to groups currently.

**Q: Can I see delivery/read receipts?**
A: Not in current version. Feature planned for future release.

**Q: Can I receive real-time messages?**
A: Yes! Run `whatsapp-cli sync` to continuously receive and store messages. Run it in the background or tmux session.

### Technical

**Q: Why Go instead of Python/Node.js?**
A: Go provides single binary distribution, better performance, and whatsmeow is written in Go.

**Q: Can I run this in Docker?**
A: Yes, but QR code authentication requires terminal access. Use `-it` flags for interactive mode.

**Q: Does it support proxy/VPN?**
A: Respects system proxy settings. Set `HTTP_PROXY` and `HTTPS_PROXY` environment variables.

**Q: Can I access messages from before installing CLI?**
A: Yes! When you run `whatsapp-cli sync`, it downloads message history from WhatsApp servers. The amount of history depends on WhatsApp's server-side retention.

### Privacy & Security

**Q: Is my data safe?**
A: All data stored locally. No cloud sync. Protect your `store/` directory.

**Q: Can WhatsApp detect/ban this?**
A: Uses official WhatsApp Web protocol. Risk is minimal but use at own discretion.

**Q: Is end-to-end encryption maintained?**
A: Yes, all messages are E2E encrypted using WhatsApp's protocol.

---

## Credits

### Authors

- **Vicente Reig** - Initial development
- Built with assistance from **Claude Code** (Anthropic)

### Dependencies

- [whatsmeow](https://github.com/tulir/whatsmeow) by Tulir Asokan - WhatsApp Web client library
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver
- [qrterminal](https://github.com/mdp/qrterminal) - QR code generation

### License

MIT License - see [LICENSE](LICENSE) file

### Support

- **Issues**: https://github.com/vicentereig/whatsapp-cli/issues
- **Discussions**: https://github.com/vicentereig/whatsapp-cli/discussions
- **Pull Requests**: Welcome! See [CONTRIBUTING.md](CONTRIBUTING.md)

### Acknowledgments

- WhatsApp for the multidevice protocol
- Tulir Asokan for the excellent whatsmeow library
- Go community for excellent tooling
- Anthropic for Claude Code

---

**Last Updated**: 2025-12-13
**Version**: 1.3.2
**Documentation**: https://github.com/vicentereig/whatsapp-cli/blob/main/README.md
