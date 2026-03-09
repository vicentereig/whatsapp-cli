package cmd

import "github.com/spf13/cobra"

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync messages continuously (run until Ctrl+C)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithApp(func() string {
			ctx, cancel := newContext(true)
			defer cancel()
			return app.Sync(ctx)
		})
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
