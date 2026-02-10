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

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import an agent from a file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]
		format, _ := cmd.Flags().GetString("format")

		// Determine format
		if format == "" {
			ext := strings.ToLower(filepath.Ext(inputFile))
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

		// Read file
		f, err := os.Open(inputFile)
		if err != nil {
			fmt.Printf("Error opening file: %v\n", err)
			return
		}
		defer f.Close()

		var a *agent.Agent

		// Parse
		switch strings.ToLower(format) {
		case "json":
			a = &agent.Agent{}
			if err := json.NewDecoder(f).Decode(a); err != nil {
				fmt.Printf("Error decoding JSON: %v\n", err)
				return
			}
		case "yaml", "yml":
			a = &agent.Agent{}
			if err := yaml.NewDecoder(f).Decode(a); err != nil {
				fmt.Printf("Error decoding YAML: %v\n", err)
				return
			}
		case "agt":
			a, err = agent.ParseAgent(f)
			if err != nil {
				fmt.Printf("Error parsing AGT: %v\n", err)
				return
			}
		default:
			fmt.Printf("Unsupported format: %s\n", format)
			return
		}

		// Determine destination path
		paths := viper.GetStringSlice("agent_paths")
		if len(paths) == 0 {
			fmt.Println("No agent paths configured in allmend.conf")
			return
		}
		destDir := paths[0] // Use the first path as default
		destFile := filepath.Join(destDir, strings.ToLower(a.Name)+".agt")

		// Write to .agt file
		out, err := os.Create(destFile)
		if err != nil {
			fmt.Printf("Error creating destination file: %v\n", err)
			return
		}
		defer out.Close()

		if err := agent.WriteAgent(out, a); err != nil {
			fmt.Printf("Error writing agent: %v\n", err)
			return
		}

		fmt.Printf("Imported agent '%s' to %s\n", a.Name, destFile)
	},
}

func init() {
	importCmd.Flags().String("format", "", "Format of the input file (json, yaml, agt)")
	agentCmd.AddCommand(importCmd)
}
