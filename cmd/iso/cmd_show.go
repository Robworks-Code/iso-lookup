package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ringo380/iso-lookup/internal/parse"
	"github.com/ringo380/iso-lookup/internal/render"
	"github.com/spf13/cobra"
)

var (
	showJSON        bool
	showInteractive bool
)

var showCmd = &cobra.Command{
	Use:   "show <reference>",
	Short: "Show metadata up front, plus a table of contents if a local file exists",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadCatalog()
		if err != nil {
			return err
		}
		rec, ok := c.Lookup(args[0])
		if !ok {
			fmt.Fprintf(os.Stderr, "no exact match for %q; closest:\n", args[0])
			fmt.Fprint(os.Stderr, render.SearchList(limit(c.Search(args[0]), 10)))
			return fmt.Errorf("not found")
		}
		if showJSON {
			return json.NewEncoder(os.Stdout).Encode(rec)
		}
		fmt.Print(render.Summary(rec))

		lib, err := loadLibrary()
		if err != nil {
			return err
		}
		path, ok := lib.Find(rec.Reference)
		if !ok {
			fmt.Print(render.NoLocalFile(rec))
			return nil
		}
		doc, err := parse.Parse(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not parse %s: %v\n", path, err)
			return nil
		}
		if showInteractive {
			return runTUI(rec, doc)
		}
		fmt.Print(render.TOC(doc))
		return nil
	},
}

func init() {
	showCmd.Flags().BoolVar(&showJSON, "json", false, "output JSON")
	showCmd.Flags().BoolVar(&showInteractive, "interactive", false, "browse in the TUI")
	rootCmd.AddCommand(showCmd)
}
