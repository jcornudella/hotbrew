package trss

import (
	"bufio"
	"encoding/json"
	"io"
)

// EncodeItems writes items as NDJSON (one JSON object per line).
func EncodeItems(w io.Writer, items []Item) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, item := range items {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	return nil
}

// EncodeDigest writes a digest as a single JSON line.
func EncodeDigest(w io.Writer, d *Digest) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(d)
}

// DecodeItems reads NDJSON items from a reader.
func DecodeItems(r io.Reader) ([]Item, error) {
	var items []Item
	scanner := bufio.NewScanner(r)

	// Increase buffer for large lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var item Item
		if err := json.Unmarshal(line, &item); err != nil {
			continue // skip malformed lines
		}
		items = append(items, item)
	}

	return items, scanner.Err()
}

// DecodeDigest reads a single digest from a reader.
func DecodeDigest(r io.Reader) (*Digest, error) {
	var d Digest
	if err := json.NewDecoder(r).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}
