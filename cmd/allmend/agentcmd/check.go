package agentcmd

import (
	"fmt"

	"github.com/SUSE/allmend/cmd/allmend/toolcmd"
	"github.com/SUSE/allmend/pkg/agent"
	"github.com/SUSE/allmend/pkg/tool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var checkCmd = &cobra.Command{
	Use:   "check [AGENT_NAME]",
	Short: "Check if required tools for an agent are available",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]

		// 1. Load Agents
		paths := viper.GetStringSlice("agent_paths")
		agentsMap, err := agent.Get(paths)
		if err != nil {
			return fmt.Errorf("error loading agents: %w", err)
		}

		targetAgent, exists := agentsMap[agentName]
		if !exists {
			return fmt.Errorf("agent '%s' not found", agentName)
		}

		// 2. Load Tools Store
		toolsPath, err := toolcmd.GetToolsFilePath()
		if err != nil {
			return fmt.Errorf("error determining tools file path: %w", err)
		}

		toolStore, err := tool.Load(toolsPath)
		if err != nil {
			return fmt.Errorf("error loading tools from %s: %w", toolsPath, err)
		}

		// 3. Check Tools using agent.LoadTools
		// This will dynamically fetch tools from servers matching the agent's requirements.
		if targetAgent.Tools == nil {
			fmt.Println("Agent has no tool requirements.")
			return nil
		}

		fmt.Printf("Checking tools for agent '%s'...\n", agentName)
		loadedTools, err := agent.LoadTools(targetAgent, toolStore)
		if err != nil {
			return fmt.Errorf("error loading tools: %w", err)
		}

		availableTools := make(map[string]bool)
		for _, t := range loadedTools {
			availableTools[t.Name()] = true
		}

		var missingRequired []string
		for _, req := range targetAgent.Tools.Required {
			if !availableTools[req.Name] {
				missingRequired = append(missingRequired, req.Name)
			}
		}

		var missingRecommended []string
		for _, rec := range targetAgent.Tools.Recommended {
			if !availableTools[rec.Name] {
				missingRecommended = append(missingRecommended, rec.Name)
			}
		}

		// 4. Report
		if len(missingRequired) > 0 {
			fmt.Printf("Error: Missing required tools for agent '%s':\n", agentName)
			for _, t := range missingRequired {
				fmt.Printf("  - %s\n", t)
			}
		}

		if len(missingRecommended) > 0 {
			fmt.Printf("Warning: Missing recommended tools for agent '%s':\n", agentName)
			for _, t := range missingRecommended {
				fmt.Printf("  - %s\n", t)
			}
		}

		if len(missingRequired) > 0 {
			return fmt.Errorf("missing required tools")
		}

		if len(missingRecommended) == 0 {
			fmt.Printf("All required and recommended tools for agent '%s' are available.\n", agentName)
		} else {
			fmt.Printf("Required tools are available, but some recommended tools are missing.\n")
		}

		return nil
	},
}

func init() {
	AgentCmd.AddCommand(checkCmd)
}
