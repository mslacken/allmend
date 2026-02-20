package agentcmd

import (
	"fmt"

	"github.com/SUSE/allmend/cmd/allmend/modelcmd"
	"github.com/SUSE/allmend/cmd/allmend/providercmd"
	"github.com/SUSE/allmend/pkg/agent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run [agent name]",
	Short: "Run an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		ctx := cmd.Context()
		paths := viper.GetStringSlice("agent_paths")
		agents, err := agent.Get(paths)
		if err != nil {
			return fmt.Errorf("Error loading agents: %v\n", err)
		}
		var targetAgent *agent.Agent
		for _, a := range agents {
			if a.Name == agentName {
				targetAgent = a
				break
			}
		}
		if targetAgent == nil {
			return fmt.Errorf("Agent '%s' not found in paths: %v\n", agentName, paths)
		}

		// Check tools
		if err := CheckTools(targetAgent); err != nil {
			return fmt.Errorf("tool check failed: %w", err)
		}

		// 2. Set runtime config

		modelName := viper.GetString("default_model")
		if m, _ := cmd.Flags().GetString("model"); m != "" {
			modelName = m
		}
		if modelName == "" {
			return fmt.Errorf("Error: No model specified and no default model configured.")
		}
		targetAgent.RuntimeModel = modelName

		modelsPath, err := modelcmd.GetModelsFilePath()
		if err != nil {
			return fmt.Errorf("Error determining models file path: %v\n", err)
		}
		targetAgent.ModelsFilePath = modelsPath

		providersPath, err := providercmd.GetProvidersFilePath()
		if err != nil {
			return fmt.Errorf("Error determining providers file path: %v\n", err)
		}
		targetAgent.ProvidersFilePath = providersPath

		// 3. Run agent
		chat, _ := cmd.Flags().GetBool("chat")
		yolo, _ := cmd.Flags().GetBool("yolo")
		hitl := !yolo
		if err := targetAgent.Run(ctx, chat, hitl); err != nil {
			return fmt.Errorf("Error running agent: %v\n", err)
		}
		return nil
	},
}

func init() {
	runCmd.Flags().StringP("model", "m", "", "Model to use for the agent")
	runCmd.Flags().BoolP("chat", "c", false, "Start an interactive chat session")
	runCmd.Flags().Bool("yolo", false, "Disable human-in-the-loop confirmation for tool calls (default is false, requiring confirmation)")
	AgentCmd.AddCommand(runCmd)
}
