package cmd

import (
	"github.com/lyyyuna/gococo/pkg/compile"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:                "build",
	DisableFlagParsing: true,
	Run:                buildAction,
}

func buildAction(cmd *cobra.Command, args []string) {
	compile.NewCompile(
		compile.WithBuild(),
		compile.WithArgs(args),
	)
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
