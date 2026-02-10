package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "allmend",
		Short: "Allmend Agent Framework CLI",
		Long:  `Allmend is a CLI tool for managing and interacting with AI agents using the .agt file format.`,
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
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		viper.SetConfigType("yaml") // Assume YAML for .conf
	} else {
		// Manual search for allmend.conf in priority order
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Priority: ./config -> $HOME/.config -> /usr/etc -> /etc
		// User specified a list: .config/allmend/allmend.conf /usr/etc/allmend/allmend.conf /etc/allmend.conf ./config/allmend.conf
		// Usually local overrides global.

		paths := []string{
			"./config/allmend.conf",
			filepath.Join(home, ".config", "allmend", "allmend.conf"),
			"/usr/etc/allmend/allmend.conf",
			"/etc/allmend.conf",
		}

		found := false
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				viper.SetConfigFile(p)
				viper.SetConfigType("yaml")
				found = true
				// fmt.Println("Using config:", p)
				break
			}
		}

		if !found {
			// Fallback to standard Viper search if no specific file found
			viper.AddConfigPath(filepath.Join(home, ".config", "allmend"))
			viper.AddConfigPath("/usr/etc/allmend")
			viper.AddConfigPath("/etc")
			viper.AddConfigPath("./config")
			viper.SetConfigName("allmend")
			viper.SetConfigType("yaml")
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// Config loaded successfully
	} else {
		// If explicit config file was set and failed, warn?
		// Or if found was true and failed.
		// fmt.Println("Error reading config:", err)
	}
}
