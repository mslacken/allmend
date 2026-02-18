package toolcmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ToolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage tools",
	Long:  `Manage and list available tools.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Please specify a subcommand like 'list'.")
	},
}

// GetToolsFilePath determines the path to the tools configuration file.
func GetToolsFilePath() (string, error) {
	if path := viper.GetString("tools_file"); path != "" {
		return path, nil
	}

	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return filepath.Join("config", "tools.conf"), nil
	}

	configDir := filepath.Dir(configFile)
	return filepath.Join(configDir, "tools.conf"), nil
}
