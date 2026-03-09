package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// optionalStr returns nil for empty strings, otherwise a pointer.
func optionalStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "List and search messages",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("requires a subcommand: list, search")
	},
}

var messagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List messages in a chat",
	RunE: func(cmd *cobra.Command, args []string) error {
		chatJID, _ := cmd.Flags().GetString("chat")
		limit, _ := cmd.Flags().GetInt("limit")
		page, _ := cmd.Flags().GetInt("page")
		return runWithApp(func() string {
			return app.ListMessages(optionalStr(chatJID), nil, limit, page)
		})
	},
}

var messagesSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search messages by text",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		limit, _ := cmd.Flags().GetInt("limit")
		page, _ := cmd.Flags().GetInt("page")
		return runWithApp(func() string {
			return app.ListMessages(nil, &query, limit, page)
		})
	},
}

func init() {
	messagesListCmd.Flags().String("chat", "", "chat JID to filter by")
	messagesListCmd.Flags().Int("limit", 20, "maximum messages to return")
	messagesListCmd.Flags().Int("page", 0, "page number")

	messagesSearchCmd.Flags().String("query", "", "search text")
	messagesSearchCmd.Flags().Int("limit", 20, "maximum messages to return")
	messagesSearchCmd.Flags().Int("page", 0, "page number")
	messagesSearchCmd.MarkFlagRequired("query")

	messagesCmd.AddCommand(messagesListCmd, messagesSearchCmd)
	rootCmd.AddCommand(messagesCmd)
}
