package main

import (
	"github.com/ringo380/iso-lookup/internal/catalog"
	"github.com/ringo380/iso-lookup/internal/config"
	"github.com/ringo380/iso-lookup/internal/library"
	"github.com/ringo380/iso-lookup/internal/parse"
)

func loadCatalog() (*catalog.Catalog, error) {
	path, err := config.CachePath()
	if err != nil {
		return nil, err
	}
	recs, err := catalog.LoadIndex(path)
	if err != nil {
		return nil, err
	}
	return catalog.New(recs), nil
}

func loadLibrary() (*library.Library, error) {
	c, err := config.Load()
	if err != nil {
		return nil, err
	}
	return library.New(c.DocsDir, c.IndexFile), nil
}

// limit truncates a record slice to at most n.
func limit(recs []catalog.Record, n int) []catalog.Record {
	if len(recs) > n {
		return recs[:n]
	}
	return recs
}

// runTUI is a temporary stub; Task 16 replaces it with the real TUI launcher.
func runTUI(rec catalog.Record, doc parse.Document) error { return nil }
