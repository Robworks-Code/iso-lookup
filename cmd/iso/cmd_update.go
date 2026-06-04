package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Robworks-Code/iso-lookup/internal/catalog"
	"github.com/Robworks-Code/iso-lookup/internal/config"
	"github.com/Robworks-Code/iso-lookup/internal/style"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download/refresh the ISO Open Data metadata index",
	Long: `Download the latest ISO Open Data catalogue and build the local index that
search and show query offline. Run this once before first use and whenever you
want fresh metadata. On failure the existing index is left untouched.`,
	Example: `  iso update`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(style.Dim.Render("Downloading ISO Open Data (this may take a moment)..."))
		client := &http.Client{Timeout: 5 * time.Minute}
		recs, err := catalog.BuildIndex(client)
		if err != nil {
			return fmt.Errorf("build index (existing index left untouched): %w", err)
		}
		path, err := config.CachePath()
		if err != nil {
			return err
		}
		if err := catalog.SaveIndex(path, recs); err != nil {
			return err
		}
		fmt.Println(style.Summary.Render(fmt.Sprintf("Indexed %d ISO deliverables -> %s", len(recs), path)))
		fmt.Println(style.Dim.Render("Data © ISO, via the ISO Open Data initiative, licensed under ODC-By 1.0."))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
