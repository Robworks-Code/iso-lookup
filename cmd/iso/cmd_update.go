package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ringo380/iso-lookup/internal/catalog"
	"github.com/ringo380/iso-lookup/internal/config"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download/refresh the ISO Open Data metadata index",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Downloading ISO Open Data (this may take a moment)...")
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
		fmt.Printf("Indexed %d ISO deliverables -> %s\n", len(recs), path)
		fmt.Println("Data © ISO, via the ISO Open Data initiative, licensed under ODC-By 1.0.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
