package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// renderRows prints rows as a table (default) or JSON (--output json).
// Each row is a slice aligned with the header. For JSON output, rows
// is marshaled as-is — pass a typed value via renderJSON instead if
// you want a richer shape.
func renderRows(w io.Writer, headers []string, rows [][]string) error {
	if outputFormat == "json" {
		objs := make([]map[string]string, len(rows))
		for i, row := range rows {
			obj := map[string]string{}
			for j, h := range headers {
				if j < len(row) {
					obj[h] = row[j]
				}
			}
			objs[i] = obj
		}
		return renderJSON(w, objs)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}

func renderJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
