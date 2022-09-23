package cmd

import "github.com/spf13/cobra"

var runCmd = &cobra.Command{
	Use:                "run",
	DisableFlagParsing: true,
	Run:                runAction,
}

func runAction(cmd *cobra.Command, args []string) {

}

func init() {
	rootCmd.AddCommand(runCmd)
}
