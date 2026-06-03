package main

import (
	"fmt"

	"github.com/Robworks-Code/iso-lookup/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or set configuration (docs folder, index file, pager)",
	Long: `Show the current configuration, or use a set-* subcommand to change it.
Settings are stored in config.json under your config directory
($XDG_CONFIG_HOME/iso-lookup or ~/.config/iso-lookup).`,
	Example: `  iso config
  iso config set-docs ~/standards
  iso config set-index ~/standards/index.yaml
  iso config set-pager "less -R"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Printf("docs_dir:   %s\nindex_file: %s\npager:      %s\n", c.DocsDir, c.IndexFile, c.Pager)
		return nil
	},
}

var configSetDocs = &cobra.Command{
	Use:   "set-docs <path>",
	Short: "Set the local docs folder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load()
		if err != nil {
			return err
		}
		c.DocsDir = args[0]
		return config.Save(c)
	},
}

var configSetIndex = &cobra.Command{
	Use:   "set-index <path>",
	Short: "Set the optional index.yaml override file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load()
		if err != nil {
			return err
		}
		c.IndexFile = args[0]
		return config.Save(c)
	},
}

var configSetPager = &cobra.Command{
	Use:   "set-pager <command>",
	Short: "Set the pager (e.g. less); empty disables paging",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load()
		if err != nil {
			return err
		}
		c.Pager = args[0]
		return config.Save(c)
	},
}

func init() {
	configCmd.AddCommand(configSetDocs, configSetIndex, configSetPager)
	rootCmd.AddCommand(configCmd)
}
