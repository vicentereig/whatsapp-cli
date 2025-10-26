# Quick Start Guide

## 1. Build

```bash
go build -o whatsapp-cli .
```

## 2. First Run - Authenticate

```bash
./whatsapp-cli auth
```

Scan the QR code with your WhatsApp mobile app (Settings → Linked Devices → Link a Device).

## 3. Try Commands

### List your chats
```bash
./whatsapp-cli chats list
```

### Search for a contact
```bash
./whatsapp-cli contacts search --query "John"
```

### View messages from a chat
```bash
# Get the JID from chats list or contacts search first
./whatsapp-cli messages list --chat 1234567890@s.whatsapp.net --limit 20
```

### Send a message
```bash
# To individual (phone number works)
./whatsapp-cli send --to 1234567890 --message "Hello!"

# To group (need full JID)
./whatsapp-cli send --to 1234567890@g.us --message "Hello group!"
```

## 4. Use with Scripts

All output is JSON:

```bash
# Extract JID from search
JID=$(./whatsapp-cli contacts search --query "Alice" | jq -r '.data[0].jid')

# Send message
./whatsapp-cli send --to "$JID" --message "Hi Alice!"
```

## 5. Use with AI/Codex

```javascript
const { execSync } = require('child_process');

// Search contacts
const result = JSON.parse(
  execSync('./whatsapp-cli contacts search --query "Team"').toString()
);

if (result.success) {
  console.log('Found contacts:', result.data);
}
```

## Storage

Session and messages stored in `./store/`:
- `whatsapp.db` - WhatsApp session (don't share!)
- `messages.db` - Message history

## Tips

- **Authentication lasts ~20 days** - no need to scan QR every time
- **Phone numbers**: Can use just the number (e.g., `1234567890`) for individuals
- **Groups**: Must use full JID (e.g., `123456789@g.us`) - get from chats list
- **JSON parsing**: All commands return JSON - use `jq` or parse programmatically

## Next Steps

See [README.md](README.md) for complete documentation.
