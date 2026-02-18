package toolcmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/SUSE/allmend/pkg/tool"
	"github.com/spf13/cobra"
)

var serverListCmd = &cobra.Command{
	Use:     "serverlist",
	Aliases: []string{"servers", "ls-servers"},
	Short:   "List configured MCP servers",
	Long:    `List all MCP servers configured in the tools definition file.`,
	Run: func(cmd *cobra.Command, args []string) {
		path, err := GetToolsFilePath()
		if err != nil {
			fmt.Printf("Error determining config file path: %v\n", err)
			return
		}

		store, err := tool.Load(path)
		if err != nil {
			fmt.Printf("Error loading configuration from %s: %v\n", path, err)
			return
		}

		if len(store.Servers) == 0 {
			fmt.Println("No servers configured.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tCONNECTION\tTOOLS COUNT")

		for _, s := range store.Servers {
			conn := s.URL
			if s.Type == "stdio" {
				conn = strings.Join(s.Command, " ")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", s.Name, s.Type, conn, len(s.Tools))
		}
		w.Flush()
	},
}

func init() {
	ToolCmd.AddCommand(serverListCmd)
}
