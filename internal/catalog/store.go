package catalog

import (
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
)

// ErrNoIndex is returned by LoadIndex when no built index exists yet.
var ErrNoIndex = errors.New("no catalog index found; run `iso update`")

// SaveIndex writes records to path as gob, creating parent dirs.
func SaveIndex(path string, recs []Record) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gob.NewEncoder(f).Encode(recs)
}

// LoadIndex reads the gob index, returning ErrNoIndex if absent.
func LoadIndex(path string) ([]Record, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, ErrNoIndex
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var recs []Record
	if err := gob.NewDecoder(f).Decode(&recs); err != nil {
		return nil, err
	}
	return recs, nil
}
