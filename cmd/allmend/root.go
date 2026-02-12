package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/SUSE/allmend/cmd/allmend/agentcmd"
	"github.com/SUSE/allmend/cmd/allmend/modelcmd"
	"github.com/SUSE/allmend/cmd/allmend/providercmd"
	"github.com/SUSE/allmend/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "allmend",
		Short: "Allmend Agent Framework CLI",
		Long:  `Allmend is a CLI tool for managing and interacting with AI agents.`,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/allmend/allmend.conf or ./config/allmend.conf)")

	rootCmd.AddCommand(agentcmd.AgentCmd)
	rootCmd.AddCommand(modelcmd.ModelCmd)
	rootCmd.AddCommand(providercmd.ProviderCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	paths := config.ConfigLocations("allmend.conf")
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			cfgFile = p
			// fmt.Println("Using config:", p)
			break
		}
	}
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		viper.SetConfigType("yaml") // Assume YAML for .conf
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("No configuration found", "paths", paths)
	}
}
