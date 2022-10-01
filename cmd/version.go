package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use: "version",
	Run: versionAction,
}

func versionAction(cmd *cobra.Command, args []string) {
	fmt.Println("0.0.1")
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
