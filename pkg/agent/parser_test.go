package agent

import (
	"strings"
	"testing"
)

const exampleAgent = `%Meta
Name: TestAgent
Version: 1.0.0
%Manifest
This is a test manifest.
It has multiple lines.

And an empty line.
%Mission
To test the parser.
`

func TestParseAgent(t *testing.T) {
	r := strings.NewReader(exampleAgent)
	agent, err := ParseAgent(r)
	if err != nil {
		t.Fatalf("ParseAgent failed: %v", err)
	}

	if agent.Name != "TestAgent" {
		t.Errorf("Expected Name 'TestAgent', got '%s'", agent.Name)
	}
	// Note: Version is in Meta struct, but our parser might put it there?
	// The parser logic puts 'Version' into agent.Meta.Version
	if agent.Meta.Version != "1.0.0" {
		t.Errorf("Expected Version '1.0.0', got '%s'", agent.Meta.Version)
	}

	expectedManifest := `This is a test manifest.
It has multiple lines.

And an empty line.`

	// Standardize newlines for comparison
	gotManifest := strings.ReplaceAll(agent.Manifest.Content, "\r\n", "\n")
	if gotManifest != expectedManifest {
		t.Errorf("Manifest content mismatch.\nExpected:\n%q\nGot:\n%q", expectedManifest, gotManifest)
	}

	if strings.TrimSpace(agent.Mission.Content) != "To test the parser." {
		t.Errorf("Mission content mismatch. Got: %q", agent.Mission.Content)
	}
}
