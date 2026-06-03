// Package parse reads local standards files and returns normalized Documents.
package parse

import "github.com/Robworks-Code/iso-lookup/internal/docmodel"

// Document is a parsed local standards file.
type Document = docmodel.Document

// Section is one chapter/segment, possibly nested.
type Section = docmodel.Section
