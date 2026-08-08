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

The CLI stores the session and messages in `./store/`:
- `whatsapp.db` - WhatsApp session (keep this file private)
- `messages.db` - Message history

## Tips

- **Authentication lasts about 20 days.** You do not need to scan the QR code each time.
- **Phone numbers**: For an individual contact, you can use the number alone, for example `1234567890`.
- **Groups**: You must use the full JID, for example `123456789@g.us`. Get the JID from the chat list.
- **JSON parsing**: Every command returns JSON. Use `jq`, or parse the output in your own code.

## Next Steps

See [README.md](README.md) for complete documentation.
