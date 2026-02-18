package toolcmd

import (
	"fmt"

	"github.com/SUSE/allmend/pkg/tool"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove [URL]",
	Short:   "Remove a tool server",
	Aliases: []string{"rm"},
	Long:    `Remove a configured tool server by its URL.`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		serverURL := args[0]

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

		if _, exists := store.Servers[serverURL]; !exists {
			fmt.Printf("Error: Server '%s' not found.\n", serverURL)
			return
		}

		delete(store.Servers, serverURL)

		if err := store.Save(); err != nil {
			fmt.Printf("Error saving tools: %v\n", err)
			return
		}

		fmt.Printf("Server '%s' removed successfully.\n", serverURL)
	},
}

func init() {
	ToolCmd.AddCommand(removeCmd)
}
