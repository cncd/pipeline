package coverage

import (
	"encoding/json"
	"io"
)

// MimeType used by coverage reports.
const MimeType = "application/coverage+json"

type (
	// Report represents a coverage report.
	Report struct {
		Timestamp int64   `json:"timestmp,omitempty"`
		Command   string  `json:"command_name,omitempty"`
		Files     []File  `json:"files"`
		Metrics   Metrics `json:"metrics"`
	}

	// File represents a coverage report for a single file.
	File struct {
		Name            string  `json:"filename"`
		Digest          string  `json:"checksum,omitempty"`
		Coverage        []*int  `json:"coverage"`
		Covered         float64 `json:"covered_percent,omitempty"`
		CoveredStrength float64 `json:"covered_strength,omitempty"`
		CoveredLines    int     `json:"covered_lines,omitempty"`
		TotalLines      int     `json:"lines_of_code"`
	}

	// Metrics represents total coverage metrics for all files.
	Metrics struct {
		Covered         float64 `json:"covered_percent"`
		CoveredStrength float64 `json:"covered_strength"`
		CoveredLines    int     `json:"covered_lines"`
		TotalLines      int     `json:"total_lines"`
	}
)

// WriteTo writes the report to io.Writer w.
func (r *Report) WriteTo(w io.Writer) (n int64, err error) {
	// TODO this should write to the multi-part document so that we can
	// include the mimetype and headers.
	return n, json.NewEncoder(w).Encode(r)
}
