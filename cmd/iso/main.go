package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "iso",
	Short: "Look up ISO security and standards documents",
	Long: `iso looks up standards by keyword or reference, prints authoritative
metadata from an offline copy of the ISO Open Data index, and reads full text
from local files when you have them.

Getting started:
  1. iso update                 download/refresh the metadata index (run once)
  2. iso config set-docs <dir>  point iso at a folder of your local standards
  3. iso search <terms>         find standards; iso show <ref> for details

Metadata is © ISO via the ISO Open Data initiative, licensed under ODC-By 1.0.
Standard body text is copyrighted and is only ever read from your own files.`,
	Example: `  iso update
  iso search "information security" --published
  iso search risk --committee "SC 27" --sort date --long
  iso show ISO/IEC 27001:2022
  iso show 27001 --json
  iso chapter 27001 5
  iso browse 27001
  iso open 9001`,
}

func main() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
