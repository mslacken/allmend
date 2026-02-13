package providercmd

import (
	"context"
	"fmt"

	"github.com/SUSE/allmend/cmd/allmend/modelcmd"
	"github.com/SUSE/allmend/pkg/model"
	"github.com/SUSE/allmend/pkg/provider"
	"github.com/spf13/cobra"
)

var ModelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage models for a provider",
}

var listCmd = &cobra.Command{
	Use:   "list [PROVIDER]",
	Short: "List available models from a provider",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		providerName := args[0]
		ctx := context.Background()

		models, err := getProviderModels(ctx, providerName)
		if err != nil {
			fmt.Printf("Error listing models: %v\n", err)
			return
		}

		if len(models) == 0 {
			fmt.Println("No models found.")
			return
		}

		for _, m := range models {
			fmt.Println(m)
		}
	},
}

var addModelCmd = &cobra.Command{
	Use:   "add [PROVIDER] [MODEL_NAME]",
	Short: "Add a model from a provider to the global model list",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		providerName := args[0]
		ctx := context.Background()

		// If a specific model is requested
		if len(args) == 2 {
			modelName := args[1]
			err := addModelToGlobalStore(providerName, modelName)
			if err != nil {
				fmt.Printf("Error adding model: %v\n", err)
			} else {
				fmt.Printf("Model '%s' added successfully.\n", modelName)
			}
			return
		}

		// If no model specified, sync all
		count, err := syncProviderModels(ctx, providerName)
		if err != nil {
			fmt.Printf("Error syncing models: %v\n", err)
			return
		}
		if count == 0 {
			fmt.Println("No new models found to add.")
		} else {
			fmt.Printf("Successfully added %d models.\n", count)
		}
	},
}

func init() {
	ProviderCmd.AddCommand(ModelCmd)
	ModelCmd.AddCommand(listCmd)
	ModelCmd.AddCommand(addModelCmd)
}

// Helper functions

func getProviderModels(ctx context.Context, providerName string) ([]string, error) {
	path, err := GetProvidersFilePath()
	if err != nil {
		return nil, fmt.Errorf("determining providers file path: %w", err)
	}

	store, err := provider.Load(path)
	if err != nil {
		return nil, fmt.Errorf("loading providers from %s: %w", path, err)
	}

	p, ok := store.Items[providerName]
	if !ok {
		return nil, fmt.Errorf("provider '%s' not found", providerName)
	}

	conn, err := p.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("connecting to provider '%s': %w", providerName, err)
	}

	models, err := conn.GetModells(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching models from provider '%s': %w", providerName, err)
	}

	return models, nil
}

func addModelToGlobalStore(providerName, modelName string) error {
	modelsPath, err := modelcmd.GetModelsFilePath()
	if err != nil {
		return fmt.Errorf("determining models file path: %w", err)
	}

	modelStore, err := model.Load(modelsPath)
	if err != nil {
		return fmt.Errorf("loading models from %s: %w", modelsPath, err)
	}

	// Check if model already exists (by name)
	// We use the modelName as the key and the Name field
	if _, exists := modelStore.Items[modelName]; exists {
		// Already exists, maybe update? For now, skip or error.
		// User might want to overwrite? Let's assume skip/error for "add".
		return fmt.Errorf("model '%s' already exists", modelName)
	}

	newModel := model.Model{
		Name:     modelName,
		Provider: providerName,
		// Type and Description could be fetched if the provider API supported it,
		// but GetModells only returns strings currently.
		Type: "chat", // Defaulting to chat? Or unknown?
	}

	modelStore.Items[modelName] = newModel
	return modelStore.Save()
}

func syncProviderModels(ctx context.Context, providerName string) (int, error) {
	availableModels, err := getProviderModels(ctx, providerName)
	if err != nil {
		return 0, err
	}

	modelsPath, err := modelcmd.GetModelsFilePath()
	if err != nil {
		return 0, fmt.Errorf("determining models file path: %w", err)
	}

	modelStore, err := model.Load(modelsPath)
	if err != nil {
		return 0, fmt.Errorf("loading models from %s: %w", modelsPath, err)
	}

	addedCount := 0
	for _, mName := range availableModels {
		if _, exists := modelStore.Items[mName]; !exists {
			newModel := model.Model{
				Name:     mName,
				Provider: providerName,
				Type:     "chat", // Default
			}
			modelStore.Items[mName] = newModel
			addedCount++
		}
	}

	if addedCount > 0 {
		if err := modelStore.Save(); err != nil {
			return 0, fmt.Errorf("saving models: %w", err)
		}
	}

	return addedCount, nil
}
