package agentcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/SUSE/allmend/pkg/agent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var showCmd = &cobra.Command{
	Use:   "show [agent-name]",
	Short: "Show details of an agent",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		paths := viper.GetStringSlice("agent_paths")
		return agent.ListNames(paths), cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		paths := viper.GetStringSlice("agent_paths")

		agents, err := agent.Get(paths)
		if err != nil {
			return err
		}
		var targetAgent *agent.Agent
		for _, a := range agents {
			if a.Name == agentName {
				targetAgent = a
				break
			}
		}

		if targetAgent == nil {
			return fmt.Errorf("Agent '%s' not found\n", agentName)
		}

		if targetAgent.SourceFile == "" {
			return fmt.Errorf("Source file for agent '%s' not found\n", agentName)
		}

		file, err := os.Open(targetAgent.SourceFile)
		if err != nil {
			return fmt.Errorf("could not open agent file %s: %w", targetAgent.SourceFile, err)
		}
		defer file.Close()

		displayFile, err := agent.ParseFileForDisplay(file)
		if err != nil {
			return fmt.Errorf("could not parse agent file %s: %w", targetAgent.SourceFile, err)
		}

		var inMetaSection bool
		for _, token := range displayFile.Tokens {
			switch {
			case token.Comment != nil:
				// Italic Grey
				fmt.Printf("\033[3m\033[90m%s\033[0m", *token.Comment)
			case token.BlockComment != nil:
				// Italic Grey
				fmt.Printf("\033[3m\033[90m%s\033[0m", *token.BlockComment)
			case token.Header != nil:
				inMetaSection = strings.EqualFold(strings.TrimSpace(*token.Header), "%meta")
				// Bold
				fmt.Printf("\033[1m%s\033[0m", *token.Header)
			case token.Line != nil:
				line := *token.Line
				if inMetaSection {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						// Bold Key
						fmt.Printf("\033[1m%s\033[0m:%s", parts[0], parts[1])
					} else {
						fmt.Print(line)
					}
				} else {
					fmt.Print(line)
				}
			case token.Newline != nil:
				fmt.Print(*token.Newline)
			}
		}
		// ensure there is a final newline if the file doesn't have one.
		fmt.Println()

		return nil
	},
}

func init() {
	AgentCmd.AddCommand(showCmd)
}
