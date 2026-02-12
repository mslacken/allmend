package agentcmd

import (
	"fmt"
	"os"

	"github.com/SUSE/allmend/pkg/agent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [agent-name]",
	Short: "Delete an agent",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		paths := viper.GetStringSlice("agent_paths")
		return agent.ListNames(paths), cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		agentName := args[0]
		paths := viper.GetStringSlice("agent_paths")

		agents, _ := agent.List(paths)
		var targetAgent *agent.Agent
		for _, a := range agents {
			if a.Name == agentName {
				targetAgent = a
				break
			}
		}

		if targetAgent != nil && targetAgent.SourceFile != "" {
			if err := os.Remove(targetAgent.SourceFile); err != nil {
				fmt.Printf("Error deleting agent '%s' (file: %s): %v\n", agentName, targetAgent.SourceFile, err)
				return
			}
			fmt.Printf("Deleted agent '%s' (file: %s)\n", agentName, targetAgent.SourceFile)
			return
		}

		fmt.Printf("Agent '%s' not found\n", agentName)
	},
}

func init() {
	AgentCmd.AddCommand(deleteCmd)
}
