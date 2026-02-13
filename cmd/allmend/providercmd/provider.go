package providercmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/SUSE/allmend/pkg/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage model providers",
	Long:  `List and manage available model providers.`,
}

// GetProvidersFilePath determines the path to the providers configuration file.
func GetProvidersFilePath() (string, error) {
	// 1. Check if configured explicitly in allmend.conf
	if path := viper.GetString("providers_file"); path != "" {
		return path, nil
	}

	// 2. Default: Same directory as allmend.conf, named "providers.conf"
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		// Fallback
		return filepath.Join("config", "providers.conf"), nil
	}

	configDir := filepath.Dir(configFile)
	return filepath.Join(configDir, "providers.conf"), nil
}

var listProvidersCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available providers",
	Run: func(cmd *cobra.Command, args []string) {
		path, err := GetProvidersFilePath()
		if err != nil {
			fmt.Printf("Error determining providers file path: %v\n", err)
			return
		}

		store, err := provider.Load(path)
		if err != nil {
			fmt.Printf("Error loading providers from %s: %v\n", path, err)
			return
		}

		providers := store.List()
		if len(providers) == 0 {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Printf("No providers configured (file %s does not exist).\n", path)
				return
			}
			fmt.Println("No providers configured.")
			return
		}

		format, _ := cmd.Flags().GetString("format")
		tmpl, err := template.New("list").Parse(format)
		if err != nil {
			fmt.Printf("Error parsing template: %v\n", err)
			return
		}

		for _, p := range providers {
			if err := tmpl.Execute(os.Stdout, p); err != nil {
				fmt.Printf("Error executing template: %v\n", err)
			}
		}
	},
}

func init() {
	listProvidersCmd.Flags().String("format", "- {{.Name}}: {{.Type}}\n", "Format string for listing providers")
	ProviderCmd.AddCommand(listProvidersCmd)
}
