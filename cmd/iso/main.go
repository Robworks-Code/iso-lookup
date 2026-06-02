package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "iso",
	Short: "Look up ISO security and standards documents",
	Long:  "iso looks up standards by keyword or reference, prints metadata from the offline ISO Open Data index, and reads full text from local files when available.",
}

func main() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
