package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "Download media attachments",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("requires a subcommand: download")
	},
}

var mediaDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download media for a message",
	RunE: func(cmd *cobra.Command, args []string) error {
		messageID, _ := cmd.Flags().GetString("message-id")
		chatJID, _ := cmd.Flags().GetString("chat")
		outputPath, _ := cmd.Flags().GetString("output")

		return runWithApp(func() string {
			ctx, cancel := newContext(false)
			defer cancel()
			return app.DownloadMedia(ctx, messageID, optionalStr(chatJID), outputPath)
		})
	},
}

func init() {
	mediaDownloadCmd.Flags().String("message-id", "", "message identifier")
	mediaDownloadCmd.Flags().String("chat", "", "chat JID (optional)")
	mediaDownloadCmd.Flags().String("output", "", "output file or directory")
	mediaDownloadCmd.MarkFlagRequired("message-id")

	mediaCmd.AddCommand(mediaDownloadCmd)
	rootCmd.AddCommand(mediaCmd)
}
