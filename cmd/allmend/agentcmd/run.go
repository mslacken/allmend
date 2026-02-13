package agentcmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/SUSE/allmend/cmd/allmend/modelcmd"
	"github.com/SUSE/allmend/cmd/allmend/providercmd"
	"github.com/SUSE/allmend/pkg/agent"
	"github.com/SUSE/allmend/pkg/model"
	"github.com/SUSE/allmend/pkg/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/console"
)

var runCmd = &cobra.Command{
	Use:   "run [agent name]",
	Short: "Run an agent in interactive mode",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		// 1. Load agent
		paths := viper.GetStringSlice("agent_paths")
		agents, err := agent.Get(paths)
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
		// 2. Load model
		modelName := viper.GetString("default_model")
		if m, _ := cmd.Flags().GetString("model"); m != "" {
			modelName = m
		}
		if modelName == "" {
			return fmt.Errorf("Error: No model specified and no default model configured.")
		}

		modelsPath, err := modelcmd.GetModelsFilePath()
		if err != nil {
			return fmt.Errorf("Error determining models file path: %v\n", err)
		}
		modelStore, err := model.Load(modelsPath)
		if err != nil {
			return fmt.Errorf("Error loading models: %v\n", err)
		}

		m, ok := modelStore.Items[modelName]
		if !ok {
			return fmt.Errorf("Error: Model '%s' not found in %s\n", modelName, modelsPath)
		}

		// 3. Load provider
		providersPath, err := providercmd.GetProvidersFilePath()
		if err != nil {
			return fmt.Errorf("Error determining providers file path: %v\n", err)
		}
		providerStore, err := provider.Load(providersPath)
		if err != nil {
			return fmt.Errorf("Error loading providers: %v\n", err)
		}

		p, ok := providerStore.Items[m.Provider]
		if !ok {
			return fmt.Errorf("Error: Provider '%s' (for model '%s') not found in %s\n", m.Provider, modelName, providersPath)
		}

		// 4. Create ADK LLM
		llm, err := p.CreateLLM(ctx, modelName)
		if err != nil {
			return fmt.Errorf("Error creating LLM: %v\n", err)
		}

		// 5. Create ADK Agent
		adkAgent, err := llmagent.New(llmagent.Config{
			Model:       llm,
			Instruction: targetAgent.Manifest.Content,
			Name:        targetAgent.Name,
		})
		if err != nil {
			return fmt.Errorf("Error creating ADK agent: %v\n", err)
		}

		// 6. Run launcher
		fmt.Printf("Running agent '%s' using model '%s'...\n", agentName, modelName)
		agentLauncher := console.NewLauncher()
		if err := agentLauncher.Run(ctx, &launcher.Config{
			AgentLoader: adkagent.NewSingleLoader(adkAgent),
		}); err != nil {
			return fmt.Errorf("Error running agent: %v\n", err)
		}
		return nil
	},
}

func init() {
	runCmd.Flags().StringP("model", "m", "", "Model to use for the agent")
	AgentCmd.AddCommand(runCmd)
}
