package main

import (
	"github.com/Robworks-Code/iso-lookup/internal/catalog"
	"github.com/Robworks-Code/iso-lookup/internal/config"
	"github.com/Robworks-Code/iso-lookup/internal/library"
	"github.com/Robworks-Code/iso-lookup/internal/parse"
	"github.com/Robworks-Code/iso-lookup/internal/tui"
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

func runTUI(rec catalog.Record, doc parse.Document) error {
	return tui.Run(rec, doc)
}
