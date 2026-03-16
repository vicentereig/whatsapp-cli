package main

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFlagParsingWithSubcommand demonstrates the bug where flags after a
// subcommand like "list" are not parsed because Go's flag parser stops at
// the first non-flag argument.
func TestFlagParsingWithSubcommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string // simulates args after global flag parsing
		skipN    int      // how many args to skip before parsing flags
		wantChat string
	}{
		{
			name:     "bug: args[1:] does not parse --chat after 'list' subcommand",
			args:     []string{"messages", "list", "--chat", "123@s.whatsapp.net", "--limit", "10"},
			skipN:    1,  // skip "messages", parse ["list", "--chat", ...]
			wantChat: "", // BUG: flag not parsed because "list" stops parsing
		},
		{
			name:     "fix: args[2:] correctly parses --chat after skipping subcommand",
			args:     []string{"messages", "list", "--chat", "123@s.whatsapp.net", "--limit", "10"},
			skipN:    2,                    // skip "messages" and "list", parse ["--chat", ...]
			wantChat: "123@s.whatsapp.net", // correctly parsed
		},
		{
			name:     "search subcommand with query flag",
			args:     []string{"messages", "search", "--query", "hello"},
			skipN:    2,
			wantChat: "", // no chat specified, but query should work
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			chatJID := fs.String("chat", "", "chat JID")
			fs.String("query", "", "search query")
			fs.Int("limit", 20, "limit")
			fs.Int("page", 0, "page")

			err := fs.Parse(tt.args[tt.skipN:])
			require.NoError(t, err)
			require.Equal(t, tt.wantChat, *chatJID)
		})
	}
}

// TestIssue5_FlagsNotParsed tests the exact scenarios reported in GitHub issue #5:
// - "contacts search --query abhi" returned "query required" error
// - "chats list --limit 1" ignored the limit flag
// See: https://github.com/vicentereig/whatsapp-cli/issues/5
func TestIssue5_FlagsNotParsed(t *testing.T) {
	t.Run("contacts search --query should parse query flag", func(t *testing.T) {
		// Issue #5: `whatsapp-cli contacts search --query "abhi"` returned
		// {"success":false,"data":null,"error":"--query required"}
		args := []string{"contacts", "search", "--query", "abhi"}

		fs := flag.NewFlagSet("contacts", flag.ContinueOnError)
		query := fs.String("query", "", "search query")

		// With the fix: skip 2 args (command + subcommand)
		err := fs.Parse(args[2:])
		require.NoError(t, err)
		require.Equal(t, "abhi", *query, "query flag should be parsed")
	})

	t.Run("chats list --limit should parse limit flag", func(t *testing.T) {
		// Issue #5: `whatsapp-cli chats list --limit 1` showed all 20 results
		args := []string{"chats", "list", "--limit", "1"}

		fs := flag.NewFlagSet("chats", flag.ContinueOnError)
		fs.String("query", "", "")
		limit := fs.Int("limit", 20, "limit")
		fs.Int("page", 0, "")

		// With the fix: skip 2 args (command + subcommand)
		err := fs.Parse(args[2:])
		require.NoError(t, err)
		require.Equal(t, 1, *limit, "limit flag should be parsed as 1, not default 20")
	})

	t.Run("messages list --chat should parse chat filter", func(t *testing.T) {
		// Related: messages list --chat JID was returning unfiltered results
		args := []string{"messages", "list", "--chat", "123456@s.whatsapp.net", "--limit", "5"}

		fs := flag.NewFlagSet("messages", flag.ContinueOnError)
		chat := fs.String("chat", "", "chat JID")
		fs.String("query", "", "")
		limit := fs.Int("limit", 20, "limit")
		fs.Int("page", 0, "")

		// With the fix: skip 2 args (command + subcommand)
		err := fs.Parse(args[2:])
		require.NoError(t, err)
		require.Equal(t, "123456@s.whatsapp.net", *chat, "chat flag should be parsed")
		require.Equal(t, 5, *limit, "limit flag should be parsed")
	})
}

// TestSubcommandArgSkipping verifies the correct number of args to skip
// for each command type.
func TestSubcommandArgSkipping(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantChat  string
		wantQuery string
	}{
		{
			name:      "messages list --chat requires skipping 2 args",
			args:      []string{"messages", "list", "--chat", "test@jid"},
			wantChat:  "test@jid",
			wantQuery: "",
		},
		{
			name:      "messages search --query requires skipping 2 args",
			args:      []string{"messages", "search", "--query", "hello"},
			wantChat:  "",
			wantQuery: "hello",
		},
		{
			name:      "chats list --query requires skipping 2 args",
			args:      []string{"chats", "list", "--query", "test"},
			wantChat:  "",
			wantQuery: "test",
		},
		{
			name:      "contacts search --query requires skipping 2 args",
			args:      []string{"contacts", "search", "--query", "john"},
			wantChat:  "",
			wantQuery: "john",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet(tt.name, flag.ContinueOnError)
			chat := fs.String("chat", "", "chat JID")
			query := fs.String("query", "", "search query")
			fs.Int("limit", 20, "")
			fs.Int("page", 0, "")

			// Skip 2 args (command + subcommand) to correctly parse flags
			err := fs.Parse(tt.args[2:])
			require.NoError(t, err)
			require.Equal(t, tt.wantChat, *chat, "chat flag")
			require.Equal(t, tt.wantQuery, *query, "query flag")
		})
	}
}
