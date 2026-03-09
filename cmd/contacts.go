package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Search contacts",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("requires a subcommand: search")
	},
}

var contactsSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search contacts by name",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		return runWithApp(func() string {
			return app.SearchContacts(query)
		})
	},
}

func init() {
	contactsSearchCmd.Flags().String("query", "", "search text")
	contactsSearchCmd.MarkFlagRequired("query")

	contactsCmd.AddCommand(contactsSearchCmd)
	rootCmd.AddCommand(contactsCmd)
}
