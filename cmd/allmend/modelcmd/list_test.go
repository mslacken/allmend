package modelcmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/SUSE/allmend/internal/testenv"
	"github.com/SUSE/allmend/pkg/model"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestModelList(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

	// Helper to capture stdout
	captureOutput := func(f func()) string {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		
		f()
		
		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		io.Copy(&buf, r)
		return buf.String()
	}

	t.Run("DefaultLocation", func(t *testing.T) {
		// Create models map
		models := map[string]model.Model{
			"GPT-4":    {Type: "llm", Provider: "openai", Description: "OpenAI GPT-4"},
			"Claude-3": {Type: "llm", Provider: "anthropic", Description: "Anthropic Claude 3"},
		}
		
		modelsBytes, err := yaml.Marshal(models)
		if err != nil {
			t.Fatalf("Failed to marshal models: %v", err)
		}
		
		// Write to config/modells.yaml in the temp environment
		// testenv sets up 'config' dir and sets viper config to 'config/allmend.conf'
		env.WriteFile("config/modells.yaml", string(modelsBytes))

		// Run
		output := captureOutput(func() {
			listModelsCmd.Run(listModelsCmd, []string{})
		})

		assert.Contains(t, output, "GPT-4")
		assert.Contains(t, output, "Claude-3")
		assert.Contains(t, output, "OpenAI GPT-4")
		assert.Contains(t, output, "openai")
		assert.Contains(t, output, "anthropic")
	})

	t.Run("ExplicitPath", func(t *testing.T) {
		models := map[string]model.Model{
			"CustomModel": {Provider: "custom", Description: "Custom Path Model"},
		}
		modelsBytes, _ := yaml.Marshal(models)
		
		// Write to a custom location in temp env
		customPath := "custom/modells.yaml"
		env.WriteFile(customPath, string(modelsBytes))

		viper.Set("models_file", env.GetPath(customPath))
		defer viper.Set("models_file", "")

		output := captureOutput(func() {
			listModelsCmd.Run(listModelsCmd, []string{})
		})

		assert.Contains(t, output, "CustomModel")
		assert.Contains(t, output, "custom")
	})
}
