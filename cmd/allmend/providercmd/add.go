package providercmd

import (
	"context"
	"fmt"

	"github.com/SUSE/allmend/internal/config"
	"github.com/SUSE/allmend/pkg/provider"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new provider",
	Long:  `Add a new provider configuration.`,
}

// Ollama variables
var (
	ollamaEndpoint string
)

var addOllamaCmd = &cobra.Command{
	Use:   "ollama [NAME]",
	Short: "Add an Ollama provider",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		config := map[string]any{
			"endpoint": ollamaEndpoint,
		}
		skipSync, _ := cmd.Flags().GetBool("no-sync")
		addProvider(name, "ollama", config, skipSync)
	},
}

// Google / Gemini variables
var (
	googleAPIKey    string
	googleProjectID string
	googleLocation  string
	googleBackend   string
)

var addGoogleCmd = &cobra.Command{
	Use:     "google [NAME]",
	Aliases: []string{"gemini"},
	Short:   "Add a Google/Gemini provider",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		apiKey := googleAPIKey
		if apiKey == "" {
			apiKey = config.GetEnvOrFile("GEMINI_API_KEY")
			if apiKey != "" {
				fmt.Println("Using API Key from environment or .env file.")
			}
		}

		config := map[string]any{
			"api_key":    apiKey,
			"project_id": googleProjectID,
			"location":   googleLocation,
			"backend":    googleBackend,
		}
		skipSync, _ := cmd.Flags().GetBool("no-sync")
		addProvider(name, "google", config, skipSync)
	},
}

// Helper function to add a provider
func addProvider(name, typeName string, config map[string]any, skipSync bool) {
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

	if _, exists := store.Items[name]; exists {
		fmt.Printf("Error: Provider '%s' already exists.\n", name)
		return
	}

	newProvider := provider.Provider{
		Name:   name,
		Type:   typeName,
		Config: config,
	}

	store.Items[name] = newProvider

	if err := store.Save(); err != nil {
		fmt.Printf("Error saving providers: %v\n", err)
		return
	}

	fmt.Printf("Provider '%s' added successfully.\n", name)

	if !skipSync {
		fmt.Printf("Syncing models for provider '%s'...\n", name)
		ctx := context.Background()
		count, err := syncProviderModels(ctx, name)
		if err != nil {
			fmt.Printf("Warning: Failed to sync models: %v\n", err)
		} else {
			fmt.Printf("Added %d models from provider '%s'.\n", count, name)
		}
	}
}

func init() {
	ProviderCmd.AddCommand(addCmd)

	addCmd.PersistentFlags().Bool("no-sync", false, "Do not automatically sync models from provider")

	// Ollama flags
	addOllamaCmd.Flags().StringVar(&ollamaEndpoint, "endpoint", "http://localhost:11434", "Ollama API endpoint")
	addCmd.AddCommand(addOllamaCmd)

	// Google flags
	addGoogleCmd.Flags().StringVar(&googleAPIKey, "api-key", "", "Google API Key (optional, defaults to GEMINI_API_KEY env var or .env file)")
	addGoogleCmd.Flags().StringVar(&googleProjectID, "project-id", "", "Google Cloud Project ID")
	addGoogleCmd.Flags().StringVar(&googleLocation, "location", "us-central1", "Google Cloud Location")
	addGoogleCmd.Flags().StringVar(&googleBackend, "backend", "gemini", "Backend type (gemini or vertex)")
	addCmd.AddCommand(addGoogleCmd)
}
