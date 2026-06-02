package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ringo380/iso-lookup/internal/config"
	"github.com/ringo380/iso-lookup/internal/parse"
	"github.com/ringo380/iso-lookup/internal/render"
	"github.com/spf13/cobra"
)

var noPager bool

var chapterCmd = &cobra.Command{
	Use:   "chapter <reference> <section>",
	Short: "Print a single chapter/segment from the local file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, want := args[0], args[1]
		c, err := loadCatalog()
		if err != nil {
			return err
		}
		rec, ok := c.Lookup(ref)
		if !ok {
			return fmt.Errorf("no match for %q", ref)
		}
		lib, err := loadLibrary()
		if err != nil {
			return err
		}
		path, ok := lib.Find(rec.Reference)
		if !ok {
			return fmt.Errorf("no local file for %s; run `iso open %s`", rec.Reference, rec.Reference)
		}
		doc, err := parse.Parse(path)
		if err != nil {
			return err
		}
		for _, s := range doc.Flatten() {
			if strings.EqualFold(s.Number, want) || strings.EqualFold(s.Title, want) {
				out := render.Chapter(s)
				return page(out)
			}
		}
		return fmt.Errorf("section %q not found; run `iso show %s` for the contents", want, rec.Reference)
	},
}

func page(text string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	fields := strings.Fields(cfg.Pager)
	if noPager || len(fields) == 0 {
		fmt.Print(text)
		return nil
	}
	c := exec.Command(fields[0], fields[1:]...)
	c.Stdin = strings.NewReader(text)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func init() {
	chapterCmd.Flags().BoolVar(&noPager, "no-pager", false, "do not pipe through a pager")
	rootCmd.AddCommand(chapterCmd)
}
