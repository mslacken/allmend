package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadAndList(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a dummy .agt file
	agtContent := `%Meta
Name: AGTAgent
Version: 1.0.0
`
	if err := os.WriteFile(filepath.Join(tempDir, "test.agt"), []byte(agtContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a dummy .json file
	jsonAgent := Agent{
		Name: "JSONAgent",
		Meta: &AgentMeta{Version: "2.0.0"},
	}
	jsonBytes, _ := json.Marshal(jsonAgent)
	if err := os.WriteFile(filepath.Join(tempDir, "test.json"), jsonBytes, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a dummy .yaml file
	yamlAgent := Agent{
		Name: "YAMLAgent",
		Meta: &AgentMeta{Version: "3.0.0"},
	}
	yamlBytes, _ := yaml.Marshal(yamlAgent)
	if err := os.WriteFile(filepath.Join(tempDir, "test.yaml"), yamlBytes, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a dummy .yml file
	ymlAgent := Agent{
		Name: "YMLAgent",
		Meta: &AgentMeta{Version: "4.0.0"},
	}
	ymlBytes, _ := yaml.Marshal(ymlAgent)
	if err := os.WriteFile(filepath.Join(tempDir, "test.yml"), ymlBytes, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a non-agent file
	if err := os.WriteFile(filepath.Join(tempDir, "ignore.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test List
	agents, errs := List([]string{tempDir})
	if len(errs) > 0 {
		t.Errorf("List returned errors: %v", errs)
	}

	if len(agents) != 4 {
		t.Errorf("Expected 4 agents, got %d", len(agents))
	}

	found := make(map[string]bool)
	for _, a := range agents {
		found[a.Name] = true
	}

	if !found["AGTAgent"] {
		t.Error("Missing AGTAgent")
	}
	if !found["JSONAgent"] {
		t.Error("Missing JSONAgent")
	}
	if !found["YAMLAgent"] {
		t.Error("Missing YAMLAgent")
	}
	if !found["YMLAgent"] {
		t.Error("Missing YMLAgent")
	}

	// Test ListNames
	names := ListNames([]string{tempDir})
	if len(names) != 4 {
		t.Errorf("Expected 4 names, got %d", len(names))
	}
	nameMap := make(map[string]bool)
	for _, n := range names {
		nameMap[n] = true
	}
	for _, expected := range []string{"AGTAgent", "JSONAgent", "YAMLAgent", "YMLAgent"} {
		if !nameMap[expected] {
			t.Errorf("Expected name %s not found", expected)
		}
	}
}
