package cmd

import "github.com/spf13/cobra"

var installCmd = &cobra.Command{
	Use:                "install",
	DisableFlagParsing: true,
	Run:                installAction,
}

func installAction(cmd *cobra.Command, args []string) {

}

func init() {
	rootCmd.AddCommand(installCmd)
}
