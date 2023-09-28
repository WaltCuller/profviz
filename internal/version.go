package internal

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	version = "0.0.0"
)

var versionOutput = fmt.Sprintf("profviz version @ v%s\n", version)

var versionCmd = &cobra.Command{
	Use:    "version",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(versionOutput)
	},
}
