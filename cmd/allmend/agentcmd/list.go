package agentcmd

import (
	"fmt"
	"os"
	"text/template"

	"github.com/SUSE/allmend/pkg/agent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all available agents",
	Long:    `Scan the configured agent paths and list all .agt, .json, .yaml, and .yml files found.`,
	Run: func(cmd *cobra.Command, args []string) {
		paths := viper.GetStringSlice("agent_paths")
		if len(paths) == 0 {
			fmt.Println("No agent paths configured in allmend.conf")
			return
		}

		format, _ := cmd.Flags().GetString("format")
		tmpl, err := template.New("list").Parse(format)
		if err != nil {
			fmt.Printf("Error parsing template: %v\n", err)
			return
		}

		fmt.Println("Searching for agents in:", paths)

		agents, errs := agent.List(paths)
		for _, err := range errs {
			fmt.Printf("Error: %v\n", err)
		}

		if len(agents) == 0 {
			if len(errs) == 0 {
				fmt.Println("No agents found.")
			}
			return
		}

		for _, a := range agents {
			if err := tmpl.Execute(os.Stdout, a); err != nil {
				fmt.Printf("Error executing template: %v\n", err)
			}
		}
	},
}

func init() {
	listCmd.Flags().String("format", "- {{.Name}} (v{{.Meta.Version}}): {{.Description}}\n", "Format string for listing agents")
	AgentCmd.AddCommand(listCmd)
}
