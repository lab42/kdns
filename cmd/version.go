package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("version: %s\n", Version)
		fmt.Printf("commit: %s\n", Commit)
		fmt.Printf("date: %s\n", Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
