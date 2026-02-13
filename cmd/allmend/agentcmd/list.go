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
	RunE: func(cmd *cobra.Command, args []string) error {
		paths := viper.GetStringSlice("agent_paths")
		if len(paths) == 0 {
			fmt.Println("No agent paths configured in allmend.conf")
			return nil
		}

		format, _ := cmd.Flags().GetString("format")
		tmpl, err := template.New("list").Parse(format)
		if err != nil {
			return fmt.Errorf("Error parsing template: %v\n", err)
		}

		fmt.Println("Searching for agents in:", paths)

		agents, err := agent.Get(paths)
		if err != nil {
			return err
		}

		for _, a := range agents {
			if err := tmpl.Execute(os.Stdout, a); err != nil {
				fmt.Printf("Error executing template: %v\n", err)
			}
		}
		return nil
	},
}

func init() {
	listCmd.Flags().String("format", "- {{.Name}} (v{{.Meta.Version}}): {{.Description}}\n", "Format string for listing agents")
	AgentCmd.AddCommand(listCmd)
}
