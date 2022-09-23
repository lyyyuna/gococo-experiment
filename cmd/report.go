package cmd

import "github.com/spf13/cobra"

var reportCmd = &cobra.Command{
	Use: "report",
	Run: reportAction,
}

func reportAction(cmd *cobra.Command, args []string) {

}

func init() {
	rootCmd.AddCommand(reportCmd)
}
