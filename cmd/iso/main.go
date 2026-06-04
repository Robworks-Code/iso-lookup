package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

// Color-control flags. By default lipgloss auto-detects the terminal (and
// disables color when stdout is piped or NO_COLOR is set); these allow an
// explicit override.
var (
	noColor    bool
	forceColor bool
)

// Build metadata, overridden via -ldflags at release time (see .goreleaser.yaml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:     "iso",
	Version: version,
	Short:   "Look up ISO security and standards documents",
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
  iso open 9001
  iso scan ./my-project`,
	// PersistentPreRunE runs before any command's RunE, so it settles the color
	// profile before the first styled string is rendered. --no-color wins over
	// --color if both are given.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		switch {
		case noColor:
			lipgloss.SetColorProfile(termenv.Ascii)
		case forceColor:
			lipgloss.SetColorProfile(termenv.ANSI256)
		}
		return nil
	},
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.BoolVar(&noColor, "no-color", false, "disable colored output")
	pf.BoolVar(&forceColor, "color", false, "force colored output even when piped")
}

func main() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	rootCmd.SetVersionTemplate("iso {{.Version}} (" + commit + ", built " + date + ")\n")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
