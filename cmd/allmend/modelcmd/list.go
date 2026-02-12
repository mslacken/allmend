package modelcmd

import (
	"fmt"
	"os"
	"text/template"

	"github.com/SUSE/allmend/pkg/model"
	"github.com/spf13/cobra"
)

var listModelsCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available models",
	Long:    `List all AI models configured in the models definition file.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, err := getModelsFilePath()
		if err != nil {
			fmt.Printf("Error determining models file path: %v\n", err)
			return
		}

		store, err := model.Load(path)
		if err != nil {
			fmt.Printf("Error loading models from %s: %v\n", path, err)
			return
		}

		models := store.List()
		if len(models) == 0 {
			fmt.Println("No models configured.")
			return
		}

		format, _ := cmd.Flags().GetString("format")
		tmpl, err := template.New("list").Parse(format)
		if err != nil {
			fmt.Printf("Error parsing template: %v\n", err)
			return
		}

		for _, m := range models {
			if err := tmpl.Execute(os.Stdout, m); err != nil {
				fmt.Printf("Error executing template: %v\n", err)
			}
		}
	},
}

func init() {
	listModelsCmd.Flags().String("format", "- {{.Name}}: {{.Description}} ({{.Type}} via {{.Provider}})\n", "Format string for listing models")
	ModelCmd.AddCommand(listModelsCmd)
}
