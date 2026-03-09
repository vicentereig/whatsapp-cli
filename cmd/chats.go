package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var chatsCmd = &cobra.Command{
	Use:   "chats",
	Short: "List chats",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("requires a subcommand: list")
	},
}

var chatsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent chats",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		limit, _ := cmd.Flags().GetInt("limit")
		page, _ := cmd.Flags().GetInt("page")
		return runWithApp(func() string {
			return app.ListChats(optionalStr(query), limit, page)
		})
	},
}

func init() {
	chatsListCmd.Flags().String("query", "", "filter chats by name")
	chatsListCmd.Flags().Int("limit", 20, "maximum chats to return")
	chatsListCmd.Flags().Int("page", 0, "page number")

	chatsCmd.AddCommand(chatsListCmd)
	rootCmd.AddCommand(chatsCmd)
}
