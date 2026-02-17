package agent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/SUSE/allmend/pkg/model"
	"github.com/SUSE/allmend/pkg/provider"
	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// Run starts the interactive agent loop.
func (agent *Agent) Run(ctx context.Context) error {
	// 1. Load model config
	if agent.ModelsFilePath == "" {
		return fmt.Errorf("ModelsFilePath is not set")
	}
	modelStore, err := model.Load(agent.ModelsFilePath)
	if err != nil {
		return fmt.Errorf("error loading models from %s: %w", agent.ModelsFilePath, err)
	}

	if agent.RuntimeModel == "" {
		return fmt.Errorf("RuntimeModel is not set")
	}
	m, ok := modelStore.Items[agent.RuntimeModel]
	if !ok {
		return fmt.Errorf("model '%s' not found in %s", agent.RuntimeModel, agent.ModelsFilePath)
	}

	// 2. Load provider config
	if agent.ProvidersFilePath == "" {
		return fmt.Errorf("ProvidersFilePath is not set")
	}
	providerStore, err := provider.Load(agent.ProvidersFilePath)
	if err != nil {
		return fmt.Errorf("error loading providers from %s: %w", agent.ProvidersFilePath, err)
	}

	p, ok := providerStore.Items[m.Provider]
	if !ok {
		return fmt.Errorf("provider '%s' (for model '%s') not found in %s", m.Provider, agent.RuntimeModel, agent.ProvidersFilePath)
	}

	// 3. Create LLM
	llm, err := p.CreateLLM(ctx, agent.RuntimeModel, m.Config)
	if err != nil {
		return fmt.Errorf("error creating LLM: %w", err)
	}

	// 4. Create ADK Agent
	adkAgent, err := llmagent.New(llmagent.Config{
		Model:       llm,
		Instruction: agent.Manifest.Content,
		Name:        agent.Name,
	})
	if err != nil {
		return fmt.Errorf("error creating ADK agent: %w", err)
	}

	sessionService := session.InMemoryService()
	session, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: agent.Name, UserID: "user",
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	agentRunner, err := runner.New(runner.Config{
		AppName:        agent.Name,
		Agent:          adkAgent,
		SessionService: sessionService,
	})
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	fmt.Printf("Agent '%s' initialized (using model '%s'). Type '/quit', '/q' or '/exit' to stop.\n", agent.Name, agent.RuntimeModel)

	// If there is an initial mission, send it first
	if agent.Mission != nil && agent.Mission.Content != "" {
		fmt.Printf("\nExecuting initial mission: %s\n", agent.Mission.Content)
		if err := runOnce(ctx, agentRunner, session.Session.ID(), agent.Mission.Content); err != nil {
			fmt.Printf("Error running mission: %v\n", err)
		}
	}

	// Handle graceful shutdown on Ctrl-C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	// Make sure to stop receiving signals when we return
	defer signal.Stop(sigChan)

	// Use a channel to signal completion of the loop
	done := make(chan struct{})
	defer close(done)

	go func() {
		interruptCount := 0
		for {
			select {
			case <-ctx.Done():
				// Context cancelled externally
				os.Stdin.Close()
				return
			case <-done:
				// Main loop finished
				return
			case <-sigChan:
				interruptCount++
				if interruptCount == 1 {
					fmt.Println("\n(To exit, press Ctrl+C again or type /quit)")
					fmt.Print(">>> ") // Reprint prompt
				} else {
					fmt.Println("\nExiting...")
					os.Exit(0)
				}
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(">>> ")
		if !scanner.Scan() {
			break
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		// Handle exit commands
		if text == "/exit" || text == "/quit" || text == "/q" {
			break
		}

		if err := runOnce(ctx, agentRunner, session.Session.ID(), text); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		// Ignore error caused by closing stdin on exit
		if errors.Is(err, os.ErrClosed) {
			return nil
		}
		return err
	}
	return nil
}

func runOnce(ctx context.Context, agentRunner *runner.Runner, sessionID string, text string) error {
	for event, err := range agentRunner.Run(ctx, "user", sessionID,
		genai.NewContentFromText(text, genai.RoleUser), adkagent.RunConfig{
			StreamingMode: adkagent.StreamingModeNone,
		}) {
		if err != nil {
			return err
		}
		if event.Content != nil {
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					fmt.Print(part.Text)
				}
			}
		}
	}
	fmt.Println()
	return nil
}
