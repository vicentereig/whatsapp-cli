package cmd

import "github.com/spf13/cobra"

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with WhatsApp (scan QR code)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWithApp(func() string {
			ctx, cancel := newContext(false)
			defer cancel()
			return app.Auth(ctx)
		})
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
}
