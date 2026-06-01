package catalog

import (
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
)

// ErrNoIndex is returned by LoadIndex when no built index exists yet.
var ErrNoIndex = errors.New("no catalog index found; run `iso update`")

// SaveIndex writes records to path as gob atomically (write-to-temp then rename),
// creating parent dirs. A mid-write failure will not corrupt the existing index.
func SaveIndex(path string, recs []Record) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "catalog-*.gob.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// On any error before rename, clean up the temp file.
	var encErr error
	func() {
		defer tmp.Close()
		encErr = gob.NewEncoder(tmp).Encode(recs)
	}()
	if encErr != nil {
		os.Remove(tmpName)
		return encErr
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
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
