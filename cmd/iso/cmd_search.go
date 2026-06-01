package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ringo380/iso-lookup/internal/render"
	"github.com/spf13/cobra"
)

var searchJSON bool

var searchCmd = &cobra.Command{
	Use:   "search <terms...>",
	Short: "Search standards by keyword",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadCatalog()
		if err != nil {
			return err
		}
		res := c.Search(strings.Join(args, " "))
		if searchJSON {
			return json.NewEncoder(os.Stdout).Encode(res)
		}
		if len(res) > 50 {
			res = res[:50]
			fmt.Fprintln(os.Stderr, "(showing first 50 matches; refine your query)")
		}
		fmt.Print(render.SearchList(res))
		return nil
	},
}

func init() {
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output JSON")
	rootCmd.AddCommand(searchCmd)
}
