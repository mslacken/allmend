package agentcmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents",
	Long:  `Manage and list available agents in the Allmend framework.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Please specify a subcommand like 'list'.")
	},
}
