package agentcmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/SUSE/allmend/internal/testenv"
	"github.com/stretchr/testify/assert"
)

func TestAgentList(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	// Create a dummy agent file
	agentContent := `%Meta
Name: TestAgent
Description: A test agent
Author: TestUser
Version: 0.1.0

%Manifest
Some manifest content.

%Mission
Some mission content.
`
	env.WriteFile("agents/test.agt", agentContent)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	// We need to ensure viper is using the config from testenv
	// testenv.New() already calls viper.ReadInConfig() with the temp config file.

	// Reset args to avoid interference from test runner flags
	oldArgs := os.Args
	os.Args = []string{"allmend", "agent", "list"}
	defer func() { os.Args = oldArgs }()

	// Execute the list command directly
	// We can't easily call listCmd.Execute() because it's a sub-command.
	// We can call listCmd.Run(listCmd, nil)
	listCmd.Run(listCmd, []string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Assertions
	assert.Contains(t, output, "TestAgent")
	assert.Contains(t, output, "v0.1.0")
	assert.Contains(t, output, "A test agent")
}

func TestAgentListNoAgents(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	listCmd.Run(listCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// It should print searching path but no agents
	// Since we set absolute path in config, it should print that path.
	// But simply checking it doesn't crash and prints nothing about agents is good.
	assert.Contains(t, output, "Searching for agents in:")
	assert.NotContains(t, output, "- ") // No list items
}

func TestAgentListCustomFormat(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	// Create a dummy agent file
	agentContent := `%Meta
Name: FormattedAgent
Description: A formatted agent
Author: TestUser
Version: 0.2.0

%Manifest
Some manifest content.

%Mission
Some mission content.
`
	env.WriteFile("agents/format.agt", agentContent)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set custom format
	originalFormat, _ := listCmd.Flags().GetString("format")
	listCmd.Flags().Set("format", "Name: {{.Name}}, Ver: {{.Meta.Version}}")
	defer listCmd.Flags().Set("format", originalFormat)

	// Execute the list command
	listCmd.Run(listCmd, []string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Assertions
	assert.Contains(t, output, "Name: FormattedAgent, Ver: 0.2.0")
	assert.NotContains(t, output, "- FormattedAgent") // Should not use default format
}
