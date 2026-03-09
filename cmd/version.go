package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CLI version information",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		escaped, _ := json.Marshal(version)
		fmt.Printf("{\"success\":true,\"data\":{\"version\":%s},\"error\":null}\n", escaped)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
