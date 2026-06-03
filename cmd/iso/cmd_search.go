package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ringo380/iso-lookup/internal/catalog"
	"github.com/ringo380/iso-lookup/internal/render"
	"github.com/spf13/cobra"
)

var (
	searchJSON      bool
	searchLong      bool
	searchCount     bool
	searchLimit     int
	searchICS       string
	searchCommittee string
	searchStatus    string
	searchYear      string
	searchPublished bool
	searchSort      string
)

var searchCmd = &cobra.Command{
	Use:   "search <terms...>",
	Short: "Search standards by keyword, with optional filters",
	Long: `Search the offline index by keyword across reference, title, and scope.

Results are ranked by relevance (reference > title > scope) and can be narrowed
with filter flags and re-ordered with --sort. Use --json for machine-readable
output or --count to print just the number of matches.`,
	Example: `  iso search "information security"
  iso search risk --committee "SC 27" --published
  iso search quality --ics 03.100 --sort date --long
  iso search 27001 --status withdrawn
  iso search cryptography --year 2023 --limit 10
  iso search "access control" --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if searchSort != "" && !catalog.ValidSortKey(searchSort) {
			return fmt.Errorf("invalid --sort %q (valid: %s)", searchSort, strings.Join(catalog.SortKeys, ", "))
		}
		c, err := loadCatalog()
		if err != nil {
			return err
		}

		res := c.Search(strings.Join(args, " "))
		res = catalog.Filter{
			ICS:           searchICS,
			Committee:     searchCommittee,
			Status:        searchStatus,
			Year:          searchYear,
			PublishedOnly: searchPublished,
		}.Apply(res)
		catalog.SortBy(res, searchSort)

		if searchCount {
			fmt.Println(len(res))
			return nil
		}
		if searchJSON {
			return json.NewEncoder(os.Stdout).Encode(res)
		}

		total := len(res)
		if searchLimit > 0 && total > searchLimit {
			res = res[:searchLimit]
			fmt.Fprintf(os.Stderr, "(showing first %d of %d matches; use --limit 0 for all)\n", searchLimit, total)
		}
		if searchLong {
			fmt.Print(render.SearchListLong(res))
		} else {
			fmt.Print(render.SearchList(res))
		}
		return nil
	},
}

func init() {
	f := searchCmd.Flags()
	f.BoolVar(&searchJSON, "json", false, "output results as JSON")
	f.BoolVarP(&searchLong, "long", "l", false, "wide listing with date and committee columns")
	f.BoolVar(&searchCount, "count", false, "print only the number of matches")
	f.IntVarP(&searchLimit, "limit", "n", 50, "maximum results to display (0 = no limit)")
	f.StringVar(&searchICS, "ics", "", "filter by ICS classification code prefix (e.g. 35.030)")
	f.StringVar(&searchCommittee, "committee", "", "filter by owning committee (substring, e.g. \"SC 27\")")
	f.StringVar(&searchStatus, "status", "", "filter by status label (substring, e.g. published, withdrawn, review)")
	f.StringVar(&searchYear, "year", "", "filter by publication year (YYYY)")
	f.BoolVar(&searchPublished, "published", false, "only currently-effective standards (exclude drafts and withdrawn)")
	f.StringVar(&searchSort, "sort", "relevance", "sort order: "+strings.Join(catalog.SortKeys, ", "))
	rootCmd.AddCommand(searchCmd)
}
