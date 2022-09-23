package cmd

import "github.com/spf13/cobra"

var buildCmd = &cobra.Command{
	Use:                "build",
	DisableFlagParsing: true,
	Run:                buildAction,
}

func buildAction(cmd *cobra.Command, args []string) {

}

func init() {
	rootCmd.AddCommand(buildCmd)
}
