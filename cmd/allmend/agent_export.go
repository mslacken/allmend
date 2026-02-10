package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SUSE/allmend/pkg/agent"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var exportCmd = &cobra.Command{
	Use:   "export [agent-name] [output-file]",
	Short: "Export an agent to a file",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		agentName := args[0]
		outputFile := args[1]
		format, _ := cmd.Flags().GetString("format")

		// Load agent
		paths := viper.GetStringSlice("agent_paths")
		var a *agent.Agent
		for _, p := range paths {
			file := filepath.Join(p, agentName+".agt")
			f, err := os.Open(file)
			if err != nil {
				continue
			}
			defer f.Close()

			a, err = agent.ParseAgent(f)
			if err == nil {
				break
			}
		}

		if a == nil {
			fmt.Printf("Agent '%s' not found\n", agentName)
			return
		}

		// Determine format
		if format == "" {
			ext := strings.ToLower(filepath.Ext(outputFile))
			switch ext {
			case ".json":
				format = "json"
			case ".yaml", ".yml":
				format = "yaml"
			case ".agt":
				format = "agt"
			default:
				fmt.Printf("Unknown file extension '%s', please specify --format\n", ext)
				return
			}
		}

		// Write to output file
		out, err := os.Create(outputFile)
		if err != nil {
			fmt.Printf("Error creating output file: %v\n", err)
			return
		}
		defer out.Close()

		switch strings.ToLower(format) {
		case "json":
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			if err := enc.Encode(a); err != nil {
				fmt.Printf("Error encoding JSON: %v\n", err)
				return
			}
		case "yaml", "yml":
			enc := yaml.NewEncoder(out)
			defer enc.Close()
			if err := enc.Encode(a); err != nil {
				fmt.Printf("Error encoding YAML: %v\n", err)
				return
			}
		case "agt":
			if err := agent.WriteAgent(out, a); err != nil {
				fmt.Printf("Error writing AGT: %v\n", err)
				return
			}
		default:
			fmt.Printf("Unsupported format: %s\n", format)
			return
		}

		fmt.Printf("Exported agent '%s' to %s\n", a.Name, outputFile)
	},
}

func init() {
	exportCmd.Flags().String("format", "", "Format of the output file (json, yaml, agt)")
	agentCmd.AddCommand(exportCmd)
}
