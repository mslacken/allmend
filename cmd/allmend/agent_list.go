package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/SUSE/allmend/pkg/agent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all available agents",
	Long:    `Scan the configured agent paths and list all .agt files found.`,
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

		for _, p := range paths {
			// Resolve relative paths if necessary (relative to where command runs)
			// Glob for *.agt
			files, err := filepath.Glob(filepath.Join(p, "*.agt"))
			if err != nil {
				fmt.Printf("Error searching in %s: %v\n", p, err)
				continue
			}

			if len(files) == 0 {
				continue
			}

			for _, file := range files {
				f, err := os.Open(file)
				if err != nil {
					fmt.Printf("Error opening %s: %v\n", file, err)
					continue
				}
				defer f.Close()

				a, err := agent.ParseAgent(f)
				if err != nil {
					fmt.Printf("Error parsing %s: %v\n", file, err)
					continue
				}

				if err := tmpl.Execute(os.Stdout, a); err != nil {
					fmt.Printf("Error executing template: %v\n", err)
				}
			}
		}
	},
}

func init() {
	listCmd.Flags().String("format", "- {{.Name}} (v{{.Meta.Version}}): {{.Description}}\n", "Format string for listing agents")
	agentCmd.AddCommand(listCmd)
}
