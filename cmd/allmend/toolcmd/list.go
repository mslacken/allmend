package toolcmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

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

		if len(store.Servers) == 0 {
			fmt.Println("No tools configured.")
			return
		}

		// Collect all tools with server info
		type toolInfo struct {
			Name        string
			Description string
			ServerName  string
			ServerType  string
			Connection  string
		}

		var items []toolInfo
		for _, s := range store.Servers {
			conn := s.URL
			if s.Type == "stdio" {
				// Reconstruct command string for display
				conn = strings.Join(s.Command, " ")
			}
			
			for _, t := range s.Tools {
				items = append(items, toolInfo{
					Name:        t.Name,
					Description: t.Description,
					ServerName:  s.Name,
					ServerType:  s.Type,
					Connection:  conn,
				})
			}
		}

		// Sort by tool name
		sort.Slice(items, func(i, j int) bool {
			return items[i].Name < items[j].Name
		})

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSERVER\tTYPE\tCOMMAND/URL\tDESCRIPTION")
		
		for _, item := range items {
			// Truncate description if too long?
			desc := item.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			
			// For http type, maybe empty connection if it's just URL which is same as name sometimes?
			// But user asked to "provide the command".
			
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", item.Name, item.ServerName, item.ServerType, item.Connection, desc)
		}
		w.Flush()
	},
}

func init() {
	// Removed format flag as we are using tabwriter now
	ToolCmd.AddCommand(listCmd)
}
