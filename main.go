package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/vicentereig/whatsapp-cli/internal/commands"
)

var (
	// version is overridden at build time via -ldflags "-X main.version=X.Y.Z"
	version = "1.3.1"
)

const (
	// defaultTimeout is the maximum duration for non-sync commands
	defaultTimeout = 5 * time.Minute
)

// optionalStr returns nil for empty strings, otherwise a pointer to the string.
// Used to convert flag values to optional parameters.
func optionalStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

const usage = `WhatsApp CLI - Command line interface for WhatsApp

Usage:
  whatsapp-cli <command> [options]

Commands:
  auth                              Authenticate with WhatsApp (scan QR code)
  sync                              Sync messages continuously (run until Ctrl+C)
  messages list [--chat JID]        List messages
  messages search --query TEXT      Search messages
  contacts search --query TEXT      Search contacts
  chats list                        List chats
  send --to RECIPIENT --message TEXT                     Send a text message
  send --to RECIPIENT --image PATH [--caption TEXT]      Send an image
  media download --message-id ID [--chat JID] [--output PATH]   Download media for a message
  version                           Print CLI version information

Global Options:
  --store DIR    Storage directory (default: ./store)

Examples:
  whatsapp-cli auth
  whatsapp-cli sync                    # Keep running to sync messages
  whatsapp-cli messages list --chat 1234567890@s.whatsapp.net --limit 20
  whatsapp-cli messages search --query "meeting"
  whatsapp-cli contacts search --query "John"
  whatsapp-cli send --to 1234567890 --message "Hello"
  whatsapp-cli send --to 1234567890@g.us --message "Hello group"
`

// extractGlobalFlags pulls --store from anywhere in the arg list,
// returning the store directory and remaining args.
func extractGlobalFlags(args []string) (string, []string) {
	storeDir := "./store"
	var remaining []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--store" && i+1 < len(args):
			storeDir = args[i+1]
			i++ // skip value
		case strings.HasPrefix(args[i], "--store="):
			storeDir = strings.TrimPrefix(args[i], "--store=")
		default:
			remaining = append(remaining, args[i])
		}
	}
	return storeDir, remaining
}

func exitJSON(msg string) {
	fmt.Fprintf(os.Stderr, `{"success":false,"data":null,"error":"%s"}`+"\n", msg)
	os.Exit(1)
}

func requireSubcommand(args []string, command string, valid []string) string {
	if len(args) < 2 {
		exitJSON(fmt.Sprintf("%s requires a subcommand: %s", command, strings.Join(valid, ", ")))
	}
	sub := args[1]
	for _, v := range valid {
		if sub == v {
			return sub
		}
	}
	exitJSON(fmt.Sprintf("unknown %s subcommand: %s (valid: %s)", command, sub, strings.Join(valid, ", ")))
	return "" // unreachable
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	// Extract --store from anywhere in args,
	// so "whatsapp-cli contacts search --store /tmp" works.
	storeDir, args := extractGlobalFlags(os.Args[1:])

	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	command := args[0]

	if command == "version" {
		fmt.Printf(`{"success":true,"data":{"version":"%s"},"error":null}
`, version)
		return
	}

	// Create app
	absStoreDir, err := filepath.Abs(storeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"success":false,"data":null,"error":"invalid store path: %v"}`+"\n", err)
		os.Exit(1)
	}
	app, err := commands.NewApp(absStoreDir, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"success":false,"data":null,"error":"Failed to initialize: %v"}
`, err)
		os.Exit(1)
	}
	defer app.Close()

	// Use different timeout for sync command
	var ctx context.Context
	var cancel context.CancelFunc
	if command == "sync" {
		// For sync, use signal-based cancellation
		ctx, cancel = context.WithCancel(context.Background())
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			cancel()
		}()
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
	}
	defer cancel()

	var result string

	switch command {
	case "auth":
		result = app.Auth(ctx)

	case "sync":
		result = app.Sync(ctx)

	case "messages":
		subcommand := requireSubcommand(args, "messages", []string{"list", "search"})
		messagesCmd := flag.NewFlagSet("messages", flag.ExitOnError)
		chatJID := messagesCmd.String("chat", "", "chat JID")
		query := messagesCmd.String("query", "", "search query")
		limit := messagesCmd.Int("limit", 20, "limit")
		page := messagesCmd.Int("page", 0, "page")
		messagesCmd.Parse(args[2:])

		switch subcommand {
		case "search":
			if *query == "" {
				exitJSON("messages search requires --query")
			}
			result = app.ListMessages(nil, query, *limit, *page)
		case "list":
			result = app.ListMessages(optionalStr(*chatJID), nil, *limit, *page)
		}

	case "contacts":
		requireSubcommand(args, "contacts", []string{"search"})
		contactsCmd := flag.NewFlagSet("contacts", flag.ExitOnError)
		query := contactsCmd.String("query", "", "search query")
		contactsCmd.Parse(args[2:])

		if *query == "" {
			exitJSON("contacts search requires --query")
		}
		result = app.SearchContacts(*query)

	case "chats":
		requireSubcommand(args, "chats", []string{"list"})
		chatsCmd := flag.NewFlagSet("chats", flag.ExitOnError)
		query := chatsCmd.String("query", "", "search query")
		limit := chatsCmd.Int("limit", 20, "limit")
		page := chatsCmd.Int("page", 0, "page")
		chatsCmd.Parse(args[2:])

		result = app.ListChats(optionalStr(*query), *limit, *page)

	case "send":
		sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
		to := sendCmd.String("to", "", "recipient")
		message := sendCmd.String("message", "", "message text")
		image := sendCmd.String("image", "", "image file path")
		caption := sendCmd.String("caption", "", "image caption")
		sendCmd.Parse(args[1:])

		if *to == "" {
			exitJSON(`--to is required`)
		}
		if *image != "" && *message != "" {
			exitJSON(`--message and --image are mutually exclusive`)
		}
		if *image != "" {
			result = app.SendImage(ctx, *to, *image, *caption)
		} else if *message != "" {
			result = app.SendMessage(ctx, *to, *message)
		} else {
			exitJSON(`--message or --image required`)
		}

	case "media":
		requireSubcommand(args, "media", []string{"download"})
		downCmd := flag.NewFlagSet("media download", flag.ExitOnError)
		messageID := downCmd.String("message-id", "", "message identifier")
		chatJID := downCmd.String("chat", "", "chat JID (optional)")
		outputPath := downCmd.String("output", "", "output file or directory")
		downCmd.Parse(args[2:])

		if *messageID == "" {
			exitJSON("--message-id required")
		}
		result = app.DownloadMedia(ctx, *messageID, optionalStr(*chatJID), *outputPath)

	default:
		fmt.Fprintf(os.Stderr, `{"success":false,"data":null,"error":"Unknown command: %s"}
`, command)
		os.Exit(1)
	}

	fmt.Println(result)
}
