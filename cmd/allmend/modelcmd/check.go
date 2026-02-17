package modelcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SUSE/allmend/pkg/model"
	"github.com/SUSE/allmend/pkg/provider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var checkCmd = &cobra.Command{
	Use:   "check [MODELNAME]",
	Short: "Check the availability and configuration of a model",
	Long:  `Check if a specific model is available in the provider and display its applied configuration.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		modelName := args[0]
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 1. Load Models
		modelsPath, err := GetModelsFilePath()
		if err != nil {
			fmt.Printf("Error determining models file path: %v\n", err)
			os.Exit(1)
		}

		modelStore, err := model.Load(modelsPath)
		if err != nil {
			fmt.Printf("Error loading models from %s: %v\n", modelsPath, err)
			os.Exit(1)
		}

		m, ok := modelStore.Items[modelName]
		if !ok {
			fmt.Printf("Model '%s' not found in configuration (%s).\n", modelName, modelsPath)
			os.Exit(1)
		}

		fmt.Printf("Checking model '%s'...\n", modelName)
		fmt.Printf("- Configuration: Found in %s\n", modelsPath)

		// 2. Load Providers
		providersPath, err := getProvidersFilePath()
		if err != nil {
			fmt.Printf("Error determining providers file path: %v\n", err)
			os.Exit(1)
		}

		providerStore, err := provider.Load(providersPath)
		if err != nil {
			fmt.Printf("Error loading providers from %s: %v\n", providersPath, err)
			os.Exit(1)
		}

		p, ok := providerStore.Items[m.Provider]
		if !ok {
			fmt.Printf("Provider '%s' (referenced by model '%s') not found in configuration (%s).\n", m.Provider, modelName, providersPath)
			os.Exit(1)
		}
		fmt.Printf("- Provider: '%s' (Type: %s) found in %s\n", m.Provider, p.Type, providersPath)

		// 3. Connect to Provider
		conn, err := p.GetConnection(ctx)
		if err != nil {
			fmt.Printf("Error connecting to provider '%s': %v\n", m.Provider, err)
			os.Exit(1)
		}
		fmt.Printf("- Connection: Established\n")

		// 4. Check Availability
		availableModels, err := conn.GetModells(ctx)
		if err != nil {
			fmt.Printf("Error listing models from provider '%s': %v\n", m.Provider, err)
			os.Exit(1)
		}

		targetModelID := m.Name
		// In the current implementation (agentcmd/run.go), the model name (key) is passed to the provider.
		// m.Type seems to be used for capability (e.g. "chat"), so we don't use it as ID unless Name is empty.

		found := false
		for _, am := range availableModels {
			if am == targetModelID {
				found = true
				break
			}
		}

		if found {
			fmt.Printf("- Availability: OK (Model ID '%s' found in provider)\n", targetModelID)

			// 5. Validation
			warnings, info, err := conn.CheckModel(ctx, targetModelID, m.Config)
			if err != nil {
				fmt.Printf("- Validation: Error checking model capabilities: %v\n", err)
			} else {
				if len(warnings) > 0 {
					fmt.Println("- Validation: Warnings found:")
					for _, w := range warnings {
						fmt.Printf("  - %s\n", w)
					}
				} else {
					fmt.Println("- Validation: OK (No obvious issues found)")
				}

				if len(info) > 0 {
					// The first info string is usually a header like "Available valid parameters (unset):"
					if info[0] == "Available valid parameters (unset):" {
						fmt.Println("- Info: Available valid parameters (unset):")
						// Basic pretty print
						currentLine := "  "
						items := info[1:]
						if len(items) > 0 {
							for _, item := range items {
								if len(currentLine)+len(item)+2 > 80 {
									fmt.Println(currentLine[:len(currentLine)-2])
									currentLine = "  " + item + ", "
								} else {
									currentLine += item + ", "
								}
							}
							if len(currentLine) > 2 {
								fmt.Println(currentLine[:len(currentLine)-2])
							}
						}
					} else {
						fmt.Println("- Info: Additional details:")
						for _, msg := range info {
							fmt.Printf("  - %s\n", msg)
						}
					}
				}
			}
		} else {
			fmt.Printf("- Availability: NOT FOUND (Model ID '%s' not found in provider list)\n", targetModelID)
			fmt.Println("  Available models:")
			for _, am := range availableModels {
				fmt.Printf("  - %s\n", am)
			}
			// We don't exit here, we might still want to show config
		}

		// 6. Display Configuration
		fmt.Println("- Applied Configuration:")
		if len(m.Config) == 0 {
			fmt.Println("  (No specific configuration set)")
		} else {
			yamlEncoder := yaml.NewEncoder(os.Stdout)
			yamlEncoder.SetIndent(2)
			if err := yamlEncoder.Encode(m.Config); err != nil {
				fmt.Printf("  Error displaying config: %v\n", err)
			}
		}
	},
}

func init() {
	ModelCmd.AddCommand(checkCmd)
}

// getProvidersFilePath determines the path to the providers configuration file.
// Duplicated from providercmd to avoid circular dependency.
func getProvidersFilePath() (string, error) {
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
