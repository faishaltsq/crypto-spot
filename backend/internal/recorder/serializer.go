package recorder

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
)

// Serialize prepares a market event payload for storage.
// It applies compression for large payloads like orderbooks.
func Serialize(event *MarketEvent, rawData interface{}) error {
	rawJSON, err := json.Marshal(rawData)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// Compress ORDERBOOK events (they are large)
	if event.EventType == EventOrderbook && len(rawJSON) > 1024 {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(rawJSON); err != nil {
			return fmt.Errorf("compress payload: %w", err)
		}
		if err := gz.Close(); err != nil {
			return fmt.Errorf("close gzip writer: %w", err)
		}
		
		// Ensure we don't accidentally bloat the payload if compression made it larger
		// (rare for JSON over 1KB, but possible for very small/random data)
		if buf.Len() < len(rawJSON) {
			event.Payload = buf.Bytes()
			event.Compressed = true
			return nil
		}
	}

	// Default to uncompressed JSON
	event.Payload = rawJSON
	event.Compressed = false
	return nil
}

// Deserialize extracts the original JSON payload from a MarketEvent.
func Deserialize(event MarketEvent) (json.RawMessage, error) {
	if !event.Compressed {
		return event.Payload, nil
	}

	buf := bytes.NewReader(event.Payload)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gz.Close()

	var decompressed bytes.Buffer
	if _, err := decompressed.ReadFrom(gz); err != nil {
		return nil, fmt.Errorf("decompress payload: %w", err)
	}

	return decompressed.Bytes(), nil
}
