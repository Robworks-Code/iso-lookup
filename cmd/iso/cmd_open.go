package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

func openCommand(goos, url string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return "xdg-open", []string{url}
	}
}

var openCmd = &cobra.Command{
	Use:   "open <reference>",
	Short: "Open the official ISO URL in your browser",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadCatalog()
		if err != nil {
			return err
		}
		rec, ok := c.Lookup(args[0])
		if !ok {
			return fmt.Errorf("no match for %q", args[0])
		}
		bin, cargs := openCommand(runtime.GOOS, rec.URL)
		fmt.Println("Opening", rec.URL)
		return exec.Command(bin, cargs...).Start()
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
}
