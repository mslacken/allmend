package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [agent-name]",
	Short: "Delete an agent",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		agentName := args[0]
		paths := viper.GetStringSlice("agent_paths")

		for _, p := range paths {
			file := filepath.Join(p, agentName+".agt")
			_, err := os.Stat(file)
			if err == nil {
				// File exists, delete it
				if err := os.Remove(file); err != nil {
					fmt.Printf("Error deleting agent '%s': %v\n", agentName, err)
					return
				}
				fmt.Printf("Deleted agent '%s'\n", agentName)
				return
			}
		}

		fmt.Printf("Agent '%s' not found\n", agentName)
	},
}

func init() {
	agentCmd.AddCommand(deleteCmd)
}
