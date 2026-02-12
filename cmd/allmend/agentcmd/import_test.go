package agentcmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/SUSE/allmend/internal/testenv"
	"github.com/stretchr/testify/assert"
)

func TestAgentImport(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	// Create a dummy JSON agent
	jsonContent := `{
		"name": "TestAgent",
		"description": "A test agent from JSON",
		"manifest": { "content": "JSON Manifest" },
		"mission": { "content": "JSON Mission" },
		"meta": { "author": "Tester", "version": "1.0.0" }
	}`
	env.WriteFile("input.json", jsonContent)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run command
	importCmd.Run(importCmd, []string{env.GetPath("input.json")})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "Imported agent 'TestAgent' to")

	// Verify file existence
	destFile := env.GetPath("agents/testagent.agt")
	assert.FileExists(t, destFile)
}
