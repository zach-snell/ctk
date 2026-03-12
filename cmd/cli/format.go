package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// outputJSON returns true if the user passed --json.
func outputJSON(cmd *cobra.Command) bool {
	j, _ := cmd.Root().PersistentFlags().GetBool("json")
	return j
}

// PrintJSON formats any Go struct as pretty JSON and prints to stdout.
func PrintJSON(data any) {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

// PrintOrJSON prints formatted output or JSON depending on the --json flag.
// The formatter func should print the human-readable output.
func PrintOrJSON(cmd *cobra.Command, data any, formatter func()) {
	if outputJSON(cmd) {
		PrintJSON(data)
		return
	}
	formatter()
}

// --- Table helpers ---

// Table is a simple tabwriter-based table printer.
type Table struct {
	w *tabwriter.Writer
}

// NewTable creates a new table with tabwriter defaults.
func NewTable() *Table {
	return &Table{
		w: tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
	}
}

// Header writes a header row (uppercased automatically).
func (t *Table) Header(cols ...string) {
	upper := make([]string, len(cols))
	for i, c := range cols {
		upper[i] = strings.ToUpper(c)
	}
	fmt.Fprintln(t.w, strings.Join(upper, "\t"))
}

// Row writes a data row.
func (t *Table) Row(vals ...string) {
	fmt.Fprintln(t.w, strings.Join(vals, "\t"))
}

// Flush flushes the underlying tabwriter.
func (t *Table) Flush() {
	t.w.Flush()
}

// --- Key-value helpers ---

// KV prints a labeled key-value pair with consistent padding.
func KV(label string, value string) {
	fmt.Printf("  %-16s%s\n", label+":", value)
}

// --- Common formatters ---

// FormatTime formats a time.Time as a short human-readable string.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04")
}

// Truncate truncates a string to maxLen and adds "..." if needed.
func Truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
