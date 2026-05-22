package ingest

import "strings"

func chunk(text string, size, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	if len(text) <= size {
		return []string{text}
	}

	// Defensive bounds: an overlap >= size would loop forever.
	if overlap < 0 {
		overlap = 0
	}

	if overlap >= size {
		overlap = size / 2
	}

	// Only accept a boundary if it lies in the last 30% of the window.
	// Without this threshold, an early "\n\n" would produce tiny
	// chunks; with it, the chunker prefers full-sized chunks.
	threshold := size * 7 / 10

	var chunks []string
	n := len(text)
	start := 0
	for start < n {
		end := start + size
		if end >= n {
			if part := strings.TrimSpace(text[start:]); part != "" {
				chunks = append(chunks, part)
			}
			break
		}

		window := text[start:end]
		switch {
		case strings.LastIndex(window, "\n\n") >= threshold:
			end = start + strings.LastIndex(window, "\n\n") + 2
		case strings.LastIndex(window, ". ") >= threshold:
			end = start + strings.LastIndex(window, ". ") + 2
		case strings.LastIndex(window, " ") >= threshold:
			end = start + strings.LastIndex(window, " ") + 1
		}

		if part := strings.TrimSpace(text[start:end]); part != "" {
			chunks = append(chunks, part)
		}
		start = end - overlap
	}

	return chunks
}
