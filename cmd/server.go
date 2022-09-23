package cmd

import "github.com/spf13/cobra"

var serverCmd = &cobra.Command{
	Use: "server",
	Run: serverAction,
}

func serverAction(cmd *cobra.Command, args []string) {

}

func init() {
	rootCmd.AddCommand(serverCmd)
}
