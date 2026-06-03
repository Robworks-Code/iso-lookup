package main

import (
	"fmt"

	"github.com/Robworks-Code/iso-lookup/internal/parse"
	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse <reference>",
	Short: "Browse a standard interactively (TUI)",
	Long: `Open an interactive, full-screen browser for a standard's sections. Requires
a local copy of the full text. Navigate with the arrow keys (or j/k) and quit
with q. Equivalent to "iso show <ref> --interactive".`,
	Example: `  iso browse 27001
  iso browse ISO/IEC 27001:2022`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadCatalog()
		if err != nil {
			return err
		}
		rec, ok := c.Lookup(args[0])
		if !ok {
			return fmt.Errorf("no match for %q", args[0])
		}
		lib, err := loadLibrary()
		if err != nil {
			return err
		}
		path, ok := lib.Find(rec.Reference)
		if !ok {
			return fmt.Errorf("no local file for %s; the TUI needs full text. Run `iso open %s`", rec.Reference, rec.Reference)
		}
		doc, err := parse.Parse(path)
		if err != nil {
			return err
		}
		return runTUI(rec, doc)
	},
}

func init() {
	rootCmd.AddCommand(browseCmd)
}
