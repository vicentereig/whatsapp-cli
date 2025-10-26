package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/vicente/whatsapp-cli/internal/commands"
)

var (
	// version is overridden at build time via -ldflags "-X main.version=X.Y.Z"
	version = "dev"
)

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
  send --to RECIPIENT --message TEXT    Send a message
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

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	// Global flags
	storeDir := flag.String("store", "./store", "storage directory")
	flag.Parse()

	// Get command
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	command := args[0]
	subcommand := ""
	if len(args) > 1 {
		subcommand = args[1]
	}

	if command == "version" {
		fmt.Printf(`{"success":true,"data":{"version":"%s"},"error":null}
`, version)
		return
	}

	// Create app
	absStoreDir, _ := filepath.Abs(*storeDir)
	app, err := commands.NewApp(absStoreDir)
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
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	}
	defer cancel()

	var result string

	switch command {
	case "auth":
		result = app.Auth(ctx)

	case "sync":
		result = app.Sync(ctx)

	case "messages":
		messagesCmd := flag.NewFlagSet("messages", flag.ExitOnError)
		chatJID := messagesCmd.String("chat", "", "chat JID")
		query := messagesCmd.String("query", "", "search query")
		limit := messagesCmd.Int("limit", 20, "limit")
		page := messagesCmd.Int("page", 0, "page")
		messagesCmd.Parse(args[1:])

		if subcommand == "search" || *query != "" {
			result = app.ListMessages(nil, query, *limit, *page)
		} else {
			var chatPtr *string
			if *chatJID != "" {
				chatPtr = chatJID
			}
			result = app.ListMessages(chatPtr, nil, *limit, *page)
		}

	case "contacts":
		contactsCmd := flag.NewFlagSet("contacts", flag.ExitOnError)
		query := contactsCmd.String("query", "", "search query")
		contactsCmd.Parse(args[1:])

		if *query == "" {
			fmt.Fprintln(os.Stderr, `{"success":false,"data":null,"error":"--query required"}`)
			os.Exit(1)
		}
		result = app.SearchContacts(*query)

	case "chats":
		chatsCmd := flag.NewFlagSet("chats", flag.ExitOnError)
		query := chatsCmd.String("query", "", "search query")
		limit := chatsCmd.Int("limit", 20, "limit")
		page := chatsCmd.Int("page", 0, "page")
		chatsCmd.Parse(args[1:])

		var queryPtr *string
		if *query != "" {
			queryPtr = query
		}
		result = app.ListChats(queryPtr, *limit, *page)

	case "send":
		sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
		to := sendCmd.String("to", "", "recipient")
		message := sendCmd.String("message", "", "message text")
		sendCmd.Parse(args[1:])

		if *to == "" || *message == "" {
			fmt.Fprintln(os.Stderr, `{"success":false,"data":null,"error":"--to and --message required"}`)
			os.Exit(1)
		}
		result = app.SendMessage(ctx, *to, *message)

	default:
		fmt.Fprintf(os.Stderr, `{"success":false,"data":null,"error":"Unknown command: %s"}
`, command)
		os.Exit(1)
	}

	fmt.Println(result)
}
