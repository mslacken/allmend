package toolcmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/SUSE/allmend/internal/testenv"
	"github.com/SUSE/allmend/pkg/tool"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestServerList(t *testing.T) {
	env := testenv.New(t)
	defer env.RemoveAll()

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

	// Setup Tools with Servers
	servers := map[string]tool.Server{
		"server1": {
			Name: "server1",
			Type: "http",
			URL:  "http://example.com/api",
		},
		"server2": {
			Name: "server2",
			Type: "stdio",
			Command: []string{"npx", "server"},
		},
	}
	config := map[string]interface{}{
		"servers": servers,
	}
	toolsBytes, _ := yaml.Marshal(config)
	toolsPath := filepath.Join(env.BaseDir, "config", "tools.conf")
	env.WriteFile("config/tools.conf", string(toolsBytes))
	viper.Set("tools_file", toolsPath)

	t.Run("ListServers", func(t *testing.T) {
		output := captureOutput(func() {
			serverListCmd.Run(serverListCmd, []string{})
		})

		// Check headers
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "TYPE")
		assert.Contains(t, output, "CONNECTION")
		assert.NotContains(t, output, "TOOLS COUNT")

		// Check server 1
		assert.Contains(t, output, "server1")
		assert.Contains(t, output, "http")
		assert.Contains(t, output, "http://example.com/api")
		// Count check removed

		// Check server 2
		assert.Contains(t, output, "server2")
		assert.Contains(t, output, "stdio")
		assert.Contains(t, output, "npx server")
	})

	t.Run("ListNoServers", func(t *testing.T) {
		// Empty config
		emptyConfig := map[string]interface{}{
			"servers": map[string]tool.Server{},
		}
		toolsBytes, _ := yaml.Marshal(emptyConfig)
		env.WriteFile("config/tools.conf", string(toolsBytes))

		output := captureOutput(func() {
			serverListCmd.Run(serverListCmd, []string{})
		})

		assert.Contains(t, output, "No servers configured.")
	})
}
