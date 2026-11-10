package subtitle

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

var vttTimeRegex = regexp.MustCompile(`(?m)^((?:\d{2}:)?\d{2}:\d{2}\.\d{3})\s*-->\s*((?:\d{2}:)?\d{2}:\d{2}\.\d{3})(?:[^\r\n]*)?$`)

func NormalizeToSRT(r io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Strip UTF-8 BOM if present
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))

	// Normalize CRLF -> LF
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// If already SRT format with standard --> timestamps, ensure commas and return
	if strings.Contains(text, "-->") {
		var out strings.Builder
		lines := strings.Split(text, "\n")
		cueIndex := 1

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "WEBVTT") || strings.HasPrefix(trimmed, "NOTE") || strings.HasPrefix(trimmed, "STYLE") {
				continue
			}

			if strings.Contains(trimmed, "-->") {
				// Normalize timestamps: replace dots with commas for SRT or vice-versa
				parts := strings.Split(trimmed, "-->")
				if len(parts) >= 2 {
					start := formatTimestamp(strings.TrimSpace(parts[0]))
					endParts := strings.Fields(strings.TrimSpace(parts[1]))
					end := formatTimestamp(endParts[0])
					out.WriteString(fmt.Sprintf("%d\n%s --> %s\n", cueIndex, start, end))
					cueIndex++
					continue
				}
			}
			out.WriteString(line + "\n")
		}
		return []byte(out.String()), nil
	}

	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("file is not valid UTF-8 text")
	}

	return raw, nil
}

func formatTimestamp(ts string) string {
	ts = strings.ReplaceAll(ts, ".", ",")
	parts := strings.Split(ts, ":")
	if len(parts) == 2 {
		return "00:" + ts
	}
	return ts
}
