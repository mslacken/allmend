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

func TestParseAgentTools(t *testing.T) {
	agt := `%Meta
Name: ToolAgent
%Tools
%Required
tool1
tool2
%Recommended
tool3
`
	r := strings.NewReader(agt)
	agent, err := ParseAgent(r)
	if err != nil {
		t.Fatalf("ParseAgent failed: %v", err)
	}

	if len(agent.Tools.Required) != 2 {
		t.Errorf("Expected 2 required tools, got %d", len(agent.Tools.Required))
	} else {
		if agent.Tools.Required[0].Name != "tool1" {
			t.Errorf("Expected first tool 'tool1', got '%s'", agent.Tools.Required[0].Name)
		}
		if agent.Tools.Required[1].Name != "tool2" {
			t.Errorf("Expected second tool 'tool2', got '%s'", agent.Tools.Required[1].Name)
		}
	}

	if len(agent.Tools.Recommended) != 1 {
		t.Errorf("Expected 1 recommended tool, got %d", len(agent.Tools.Recommended))
	} else {
		if agent.Tools.Recommended[0].Name != "tool3" {
			t.Errorf("Expected recommended tool 'tool3', got '%s'", agent.Tools.Recommended[0].Name)
		}
	}
}

func TestParseSubAgents(t *testing.T) {
	agt := `%Meta
Name: Parent
%Subagent
Child1
Description for Child1
%Tools
%Required
child_tool
%Subagent
Child2
`
	r := strings.NewReader(agt)
	agent, err := ParseAgent(r)
	if err != nil {
		t.Fatalf("ParseAgent failed: %v", err)
	}

	if len(agent.SubAgents) != 2 {
		t.Fatalf("Expected 2 sub-agents, got %d", len(agent.SubAgents))
	}

	sub1 := agent.SubAgents[0]
	if sub1.Name != "Child1" {
		t.Errorf("Expected sub1 name 'Child1', got '%s'", sub1.Name)
	}
	if sub1.Description != "Description for Child1" {
		t.Errorf("Expected sub1 description 'Description for Child1', got '%s'", sub1.Description)
	}
	if len(sub1.Tools.Required) != 1 || sub1.Tools.Required[0].Name != "child_tool" {
		t.Errorf("Sub1 tool mismatch")
	}

	sub2 := agent.SubAgents[1]
	if sub2.Name != "Child2" {
		t.Errorf("Expected sub2 name 'Child2', got '%s'", sub2.Name)
	}
}
