package agent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/SUSE/allmend/pkg/model"
	"github.com/SUSE/allmend/pkg/provider"
	"github.com/SUSE/allmend/pkg/tool"
	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	adkmodel "google.golang.org/adk/model"
	adktool "google.golang.org/adk/tool"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// SessionUsage tracks the total tokens and time used in a session.
type SessionUsage struct {
	Prompt     int32
	Candidates int32
	Total      int32
	Runtime    time.Duration
}

func (u *SessionUsage) Add(other SessionUsage) {
	u.Prompt += other.Prompt
	u.Candidates += other.Candidates
	u.Total += other.Total
	u.Runtime += other.Runtime
}

func (u *SessionUsage) Print() {
	fmt.Printf("Total Runtime: %v | Total Tokens: %d (Prompt: %d, Candidates: %d)\n",
		u.Runtime.Round(time.Millisecond), u.Total, u.Prompt, u.Candidates)
}

// Run starts the agent. If chat is true, it enters an interactive loop.
func (agent *Agent) Run(ctx context.Context, chat bool, hitl bool) error {
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

	// Load Tools
	// We need to determine tools file path similar to other configs.
	// For now, let's assume standard location or reuse logic if available.
	// But agent struct doesn't have ToolsFilePath.
	// We can use viper or default location.
	// Let's assume standard location "config/tools.conf" relative to base if not provided.
	// Or maybe inject it?
	// For now, let's try to load from standard location.
	toolsPath := "config/tools.conf" // Fallback
	if agent.ModelsFilePath != "" {
		// Try to deduce from models path
		dir := filepath.Dir(agent.ModelsFilePath)
		toolsPath = filepath.Join(dir, "tools.conf")
	}

	toolStore, err := tool.Load(toolsPath)
	if err != nil {
		// Log warning or just continue without tools?
		fmt.Printf("Warning: Failed to load tools from %s: %v\n", toolsPath, err)
	}
	
	var agentTools []adktool.Tool
	if toolStore != nil {
		agentTools, err = LoadTools(agent, toolStore)
		if err != nil {
			fmt.Printf("Warning: Failed to load agent tools: %v\n", err)
		}
	}

	// Initialize SubAgents
	for _, sub := range agent.SubAgents {
		// Inherit Manifest if not set
		if sub.Manifest == nil {
			sub.Manifest = agent.Manifest
		} else if sub.Manifest.Content == "" && agent.Manifest != nil {
			sub.Manifest.Content = agent.Manifest.Content
		}

		instruction := ""
		if sub.Manifest != nil {
			instruction = sub.Manifest.Content
		}

		// Load tools for sub-agent
		var subTools []adktool.Tool
		if toolStore != nil {
			subTools, err = LoadTools(sub, toolStore)
			if err != nil {
				fmt.Printf("Warning: Failed to load tools for sub-agent %s: %v\n", sub.Name, err)
			}
		}

		// Config for sub-agent
		subConfig := llmagent.Config{
			Model:       llm,
			Instruction: instruction,
			Name:        sub.Name,
			Tools:       subTools,
		}

		if hitl {
			subConfig.BeforeToolCallbacks = []llmagent.BeforeToolCallback{
				func(ctx adktool.Context, t adktool.Tool, args map[string]any) (map[string]any, error) {
					toolType := "tool"
					needsConfirmation := true // Default for tools

					if st, ok := t.(*SubAgentTool); ok {
						toolType = "subagent"
						needsConfirmation = st.Confirmation()
					} else if mt, ok := t.(*MCPTool); ok {
						if mt.NoConfirmation() {
							needsConfirmation = false
						}
					}

					if !needsConfirmation {
						return nil, nil // Proceed without asking
					}

					fmt.Printf("\n[SubAgent: %s] Requesting to run %s '%s' with args: %v\n", sub.Name, toolType, t.Name(), args)
					fmt.Print("Allow execution? (y/n): ")
					
					var response string
					fmt.Scanln(&response)
					response = strings.ToLower(strings.TrimSpace(response))
					
					if response == "y" || response == "yes" {
						return nil, nil // Proceed
					}
					
					fmt.Println("Tool execution denied by user.")
					return nil, fmt.Errorf("tool execution denied by user")
				},
			}
		}

		// Create sub-agent
		subAdkAgent, err := llmagent.New(subConfig)
		if err != nil {
			return fmt.Errorf("error creating sub-agent %s: %w", sub.Name, err)
		}
		
		subTool, err := NewSubAgentTool(subAdkAgent, sub.Name, sub.Description, sub.Confirmation)
		if err != nil {
			return fmt.Errorf("error wrapping sub-agent %s as tool: %w", sub.Name, err)
		}
		// Wrap with logger
		loggedSubTool := &subAgentLogger{SubAgentTool: subTool}
		agentTools = append(agentTools, loggedSubTool)
	}

	// 4. Create ADK Agent
	adkConfig := llmagent.Config{
		Model:       llm,
		Instruction: agent.Manifest.Content,
		Name:        agent.Name,
		Tools:       agentTools,
	}

	if hitl {
		adkConfig.BeforeToolCallbacks = []llmagent.BeforeToolCallback{
			func(ctx adktool.Context, t adktool.Tool, args map[string]any) (map[string]any, error) {
				toolType := "tool"
				needsConfirmation := true // Default for tools

				if st, ok := t.(*SubAgentTool); ok {
					toolType = "subagent"
					needsConfirmation = st.Confirmation()
				} else if sl, ok := t.(*subAgentLogger); ok {
					toolType = "subagent"
					needsConfirmation = sl.Confirmation()
				} else if mt, ok := t.(*MCPTool); ok {
					if mt.NoConfirmation() {
						needsConfirmation = false
					}
				}

				if !needsConfirmation {
					return nil, nil // Proceed without asking
				}

				fmt.Printf("\n[Agent: %s] Requesting to run %s '%s' with args: %v\n", agent.Name, toolType, t.Name(), args)
				fmt.Print("Allow execution? (y/n): ")
				
				var response string
				fmt.Scanln(&response)
				response = strings.ToLower(strings.TrimSpace(response))
				
				if response == "y" || response == "yes" {
					return nil, nil // Proceed
				}
				
				fmt.Println("Tool execution denied by user.")
				return nil, fmt.Errorf("tool execution denied by user")
			},
		}
	}

	adkAgent, err := llmagent.New(adkConfig)
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

	totalUsage := &SessionUsage{}

	if chat {
		fmt.Printf("Agent '%s' initialized (using model '%s'). Type '/quit', '/q' or '/exit' to stop.\n", agent.Name, agent.RuntimeModel)
	}

	// If there is an initial mission, send it first
	if agent.Mission != nil && agent.Mission.Content != "" {
		if chat {
			fmt.Printf("\nExecuting initial mission: %s\n", agent.Mission.Content)
		}
		usage, err := runOnce(ctx, agentRunner, session.Session.ID(), agent.Mission.Content)
		if err != nil {
			fmt.Printf("Error running mission: %v\n", err)
		}
		totalUsage.Add(usage)
	}

	if !chat {
		totalUsage.Print()
		return nil
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
					totalUsage.Print()
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

		// Handle usage commands
		if text == "/usage" || text == "/tokens" {
			totalUsage.Print()
			continue
		}

		usage, err := runOnce(ctx, agentRunner, session.Session.ID(), text)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		totalUsage.Add(usage)
	}

	totalUsage.Print()

	if err := scanner.Err(); err != nil {
		// Ignore error caused by closing stdin on exit
		if errors.Is(err, os.ErrClosed) {
			return nil
		}
		return err
	}
	return nil
}

func runOnce(ctx context.Context, agentRunner *runner.Runner, sessionID string, text string) (SessionUsage, error) {
	startTime := time.Now()
	var usage SessionUsage

	for event, err := range agentRunner.Run(ctx, "user", sessionID,
		genai.NewContentFromText(text, genai.RoleUser), adkagent.RunConfig{
			StreamingMode: adkagent.StreamingModeNone,
		}) {
		if err != nil {
			return usage, err
		}
		if event.UsageMetadata != nil {
			usage.Prompt += event.UsageMetadata.PromptTokenCount
			usage.Candidates += event.UsageMetadata.CandidatesTokenCount
			usage.Total += event.UsageMetadata.TotalTokenCount
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
	usage.Runtime = time.Since(startTime)
	return usage, nil
}

// subAgentLogger wraps a SubAgentTool to add logging.
type subAgentLogger struct {
	*SubAgentTool
}

func (s *subAgentLogger) Run(ctx adktool.Context, args any) (map[string]any, error) {
	instruction := ""
	if m, ok := args.(map[string]any); ok {
		instruction, _ = m["instruction"].(string)
	}
	fmt.Printf("\n[System] Calling sub-agent '%s' with instruction: %s\n", s.Name(), instruction)
	
	res, err := s.SubAgentTool.Run(ctx, args)
	
	if err != nil {
		fmt.Printf("[System] Sub-agent '%s' execution failed: %v\n", s.Name(), err)
		return nil, err
	}

	// Extract content for display
	content := "no content"
	if res != nil {
		if c, ok := res["content"]; ok {
			content = fmt.Sprintf("%v", c)
		}
	}
	fmt.Printf("[System] Sub-agent '%s' result sent to agent:\n%s\n", s.Name(), content)
	
	return res, nil
}

func (s *subAgentLogger) ProcessRequest(ctx adktool.Context, req *adkmodel.LLMRequest) error {
	// Register the wrapper 's' as the tool instance so the runner calls our Run method
	return RegisterTool(req, s.SubAgentTool.Declaration(), s)
}
