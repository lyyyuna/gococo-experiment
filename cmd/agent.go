package cmd

import "github.com/spf13/cobra"

var agentCmd = &cobra.Command{
	Use: "agent",
	Run: agentAction,
}

func agentAction(cmd *cobra.Command, args []string) {

}

func init() {
	rootCmd.AddCommand(agentCmd)
}
