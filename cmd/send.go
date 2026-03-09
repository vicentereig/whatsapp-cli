package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a message or image",
	RunE: func(cmd *cobra.Command, args []string) error {
		to, _ := cmd.Flags().GetString("to")
		message, _ := cmd.Flags().GetString("message")
		image, _ := cmd.Flags().GetString("image")
		caption, _ := cmd.Flags().GetString("caption")

		// Validate before any side effects (no app init yet)
		if image != "" && message != "" {
			return fmt.Errorf("--message and --image are mutually exclusive")
		}
		if image == "" && message == "" {
			return fmt.Errorf("--message or --image required")
		}

		return runWithApp(func() string {
			ctx, cancel := newContext(false)
			defer cancel()
			if image != "" {
				return app.SendImage(ctx, to, image, caption)
			}
			return app.SendMessage(ctx, to, message)
		})
	},
}

func init() {
	sendCmd.Flags().String("to", "", "recipient JID or phone number")
	sendCmd.Flags().String("message", "", "message text")
	sendCmd.Flags().String("image", "", "image file path")
	sendCmd.Flags().String("caption", "", "image caption")
	sendCmd.MarkFlagRequired("to")

	rootCmd.AddCommand(sendCmd)
}
