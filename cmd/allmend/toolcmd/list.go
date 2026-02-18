package toolcmd

import (
	"fmt"
	"os"
	"text/template"

	"github.com/SUSE/allmend/pkg/tool"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available tools",
	Long:    `List all tools configured in the tools definition file.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, err := GetToolsFilePath()
		if err != nil {
			fmt.Printf("Error determining tools file path: %v\n", err)
			return
		}

		store, err := tool.Load(path)
		if err != nil {
			fmt.Printf("Error loading tools from %s: %v\n", path, err)
			return
		}

		tools := store.List()
		if len(tools) == 0 {
			fmt.Println("No tools configured.")
			return
		}

		format, _ := cmd.Flags().GetString("format")
		tmpl, err := template.New("list").Parse(format)
		if err != nil {
			fmt.Printf("Error parsing template: %v\n", err)
			return
		}

		for _, t := range tools {
			if err := tmpl.Execute(os.Stdout, t); err != nil {
				fmt.Printf("Error executing template: %v\n", err)
			}
		}
	},
}

func init() {
	listCmd.Flags().String("format", "- {{.Name}}: {{.Description}}\n", "Format string for listing tools")
	ToolCmd.AddCommand(listCmd)
}
