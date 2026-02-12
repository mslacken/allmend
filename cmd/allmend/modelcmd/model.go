package modelcmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ModelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage AI models",
	Long:  `Manage the available AI models for Allmend.`,
}

// getModelsFilePath determines the path to the models YAML file.
func getModelsFilePath() (string, error) {
	// 1. Check if configured explicitly in allmend.conf
	if path := viper.GetString("models_file"); path != "" {
		return path, nil
	}
	if path := viper.GetString("modells_file"); path != "" {
		return path, nil
	}

	// 2. Default: Same directory as allmend.conf, named "modells.yaml"
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		// Fallback: Check standard locations if viper hasn't found one yet
		return filepath.Join("config", "modells.yaml"), nil
	}

	configDir := filepath.Dir(configFile)
	return filepath.Join(configDir, "modells.yaml"), nil
}
