// Package out contains small output formatting helpers.
package out

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON writes v as indented JSON to stdout.
func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Status renders an open/closed marker.
func Status(open bool) string {
	if open {
		return "OPEN"
	}
	return "closed"
}

// Table prints a simple aligned two-column table.
func Table(rows [][2]string) {
	if len(rows) == 0 {
		return
	}
	width := 0
	for _, r := range rows {
		if len(r[0]) > width {
			width = len(r[0])
		}
	}
	for _, r := range rows {
		fmt.Printf("  %-*s  %s\n", width, r[0], r[1])
	}
}
