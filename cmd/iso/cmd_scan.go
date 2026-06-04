package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Robworks-Code/iso-lookup/internal/catalog"
	"github.com/Robworks-Code/iso-lookup/internal/render"
	"github.com/Robworks-Code/iso-lookup/internal/scan"
	"github.com/Robworks-Code/iso-lookup/internal/tui"
	"github.com/spf13/cobra"
)

var (
	scanJSON        bool
	scanLong        bool
	scanGroupBy     string
	scanCategory    string
	scanComp        string
	scanDiscover    bool
	scanDrafts      bool
	scanLimit       int
	scanDepth       int
	scanSort        string
	scanInteractive bool
)

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan a folder, detect its stack, and recommend relevant ISO standards",
	Long: `Scan a project folder, detect its technology stack, and advise which current
ISO standards are relevant — grouped into a report you can reshape.

Detection reads marker files (go.mod, package.json, Dockerfile, *.tf, CI configs,
…) and, where present, their dependency lists, mapping them to concerns such as
information security, privacy, AI, software lifecycle, and accessibility. Each
concern maps to a curated set of anchor standards resolved from the offline
index; --discover broadens the set via catalog search.

ISO standards address domains, not specific products, so recommendations are
advisory starting points, not a compliance checklist.`,
	Example: `  iso scan .
  iso scan ./service --discover --long
  iso scan . --group-by category
  iso scan . --component openai --json
  iso scan stack .
  iso scan why security .`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rep, err := runScan(pathArg(args))
		if err != nil {
			return err
		}
		if scanJSON {
			return json.NewEncoder(os.Stdout).Encode(rep)
		}
		if scanInteractive {
			if len(rep.Groups) == 0 {
				fmt.Print(render.ScanReport(rep, scanLong))
				return nil
			}
			return tui.RunScan(rep)
		}
		noteTruncatedGroups(rep)
		fmt.Print(render.ScanReport(rep, scanLong))
		return nil
	},
}

var scanStackCmd = &cobra.Command{
	Use:   "stack [path]",
	Short: "Show only the detected components/stack (no recommendations)",
	Example: `  iso scan stack .
  iso scan stack ./service --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		det, err := scan.Detect(pathArg(args), scanOptions())
		if err != nil {
			return err
		}
		if scanJSON {
			return json.NewEncoder(os.Stdout).Encode(det)
		}
		fmt.Print(render.ScanStack(det))
		return nil
	},
}

var scanWhyCmd = &cobra.Command{
	Use:   "why <term> [path]",
	Short: "Explain which standards a component or category drove, and why",
	Long: `Explain the recommendations for a single component, category, or concern.
The term is matched case-insensitively against group headers first, then against
the components and concerns behind each standard.`,
	Example: `  iso scan why security .
  iso scan why openai ./service
  iso scan why "Artificial Intelligence" .`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		term := args[0]
		path := "."
		if len(args) == 2 {
			path = args[1]
		}
		g, ok, err := scanWhy(term, path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("no detected component, category, or concern matches %q", term)
		}
		if scanJSON {
			return json.NewEncoder(os.Stdout).Encode(g)
		}
		fmt.Print(render.ScanWhy(g, scanLong))
		return nil
	},
}

// scanWhy finds the most relevant group for a term. It prefers a domain-category
// header match (so "devops" returns the DevOps category), then a component
// header match, then any component/concern hit — checking a category-grouped and
// a component-grouped build of the same detection.
func scanWhy(term, path string) (scan.Group, bool, error) {
	det, c, err := detect(path)
	if err != nil {
		return scan.Group{}, false, err
	}
	catRep := buildReport(det, c, scan.GroupByCategory)
	compRep := buildReport(det, c, scan.GroupByComponent)
	// A term naming a concern or category (e.g. "ai", "devops") targets that
	// category exactly, before any fuzzy substring matching.
	if cat, ok := scan.CategoryForTerm(term); ok {
		if g, ok := catRep.HeaderGroup(cat); ok {
			return g, true, nil
		}
	}
	if g, ok := compRep.HeaderGroup(term); ok {
		return g, true, nil
	}
	if g, ok := catRep.HeaderGroup(term); ok {
		return g, true, nil
	}
	if g, ok := compRep.FindGroup(term); ok {
		return g, true, nil
	}
	if g, ok := catRep.FindGroup(term); ok {
		return g, true, nil
	}
	return scan.Group{}, false, nil
}

// runScan performs detection and builds a report, applying the shared flags. It
// is used by the root scan command.
func runScan(path string) (scan.Report, error) {
	if scanSort != "" && !catalog.ValidSortKey(scanSort) {
		return scan.Report{}, fmt.Errorf("invalid --sort %q (valid: %s)", scanSort, strings.Join(catalog.SortKeys, ", "))
	}
	if !scan.ValidGroupBy(scanGroupBy) {
		return scan.Report{}, fmt.Errorf("invalid --group-by %q (valid: %s)", scanGroupBy, strings.Join(scan.GroupByKeys, ", "))
	}
	det, c, err := detect(path)
	if err != nil {
		return scan.Report{}, err
	}
	return buildReport(det, c, scanGroupBy), nil
}

// detect loads the catalog and scans the folder, returning both so a report can
// be built (possibly more than once, for different groupings).
func detect(path string) (scan.Detection, *catalog.Catalog, error) {
	c, err := loadCatalog()
	if err != nil {
		return scan.Detection{}, nil, err
	}
	det, err := scan.Detect(path, scanOptions())
	if err != nil {
		return scan.Detection{}, nil, err
	}
	return det, c, nil
}

// buildReport applies the shared build flags with an explicit grouping.
func buildReport(det scan.Detection, c *catalog.Catalog, groupBy string) scan.Report {
	return scan.Build(det, c, scan.BuildOptions{
		Discover:      scanDiscover,
		IncludeDrafts: scanDrafts,
		GroupBy:       groupBy,
		Sort:          scanSort,
		Category:      scanCategory,
		Component:     scanComp,
		LimitPerGroup: scanLimit,
	})
}

func scanOptions() scan.Options {
	opts := scan.DefaultOptions()
	opts.MaxDepth = scanDepth
	return opts
}

func pathArg(args []string) string {
	if len(args) == 1 {
		return args[0]
	}
	return "."
}

// noteTruncatedGroups prints a stderr notice when --limit hid recommendations,
// mirroring the search command's "showing first N" convention.
func noteTruncatedGroups(rep scan.Report) {
	if scanLimit <= 0 {
		return
	}
	hidden := 0
	for _, g := range rep.Groups {
		if g.Total > len(g.Recommendations) {
			hidden += g.Total - len(g.Recommendations)
		}
	}
	if hidden > 0 {
		fmt.Fprintf(os.Stderr, "(--limit %d hides %d %s; use --limit 0 for all)\n", scanLimit, hidden, plural(hidden, "standard"))
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

func init() {
	f := scanCmd.PersistentFlags()
	f.BoolVar(&scanJSON, "json", false, "output as JSON")
	f.BoolVarP(&scanLong, "long", "l", false, "wide listing with date and committee columns")
	f.StringVar(&scanGroupBy, "group-by", scan.GroupByComponent, "group standards by: "+strings.Join(scan.GroupByKeys, ", "))
	f.StringVar(&scanCategory, "category", "", "keep only groups/standards matching this category (substring)")
	f.StringVar(&scanComp, "component", "", "keep only standards driven by a matching component (substring)")
	f.BoolVar(&scanDiscover, "discover", false, "broaden recommendations via catalog search (lower confidence)")
	f.BoolVar(&scanDrafts, "include-drafts", false, "include drafts and withdrawn standards (default: current only)")
	f.IntVarP(&scanLimit, "limit", "n", 0, "maximum standards per group (0 = no limit)")
	f.IntVar(&scanDepth, "depth", 6, "maximum directory depth to scan (0 = unlimited)")
	f.StringVar(&scanSort, "sort", "relevance", "sort within each group: "+strings.Join(catalog.SortKeys, ", "))
	scanCmd.Flags().BoolVarP(&scanInteractive, "interactive", "i", false, "browse the report in an interactive TUI")

	scanCmd.AddCommand(scanStackCmd, scanWhyCmd)
	rootCmd.AddCommand(scanCmd)
}
