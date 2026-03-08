package tools

import (
	"testing"
	"unicode/utf8"
)

// FuzzParseSSEResponse tests SSE parsing with arbitrary input to catch
// panics, out-of-bounds errors, and unexpected behavior.
func FuzzParseSSEResponse(f *testing.F) {
	f.Add([]byte("data: {\"key\": \"value\"}\n"))
	f.Add([]byte("data: invalid json\n"))
	f.Add([]byte(""))
	f.Add([]byte("data: {\"a\":1}\ndata: {\"b\":2}\n"))
	f.Add([]byte("event: test\ndata: {}\n\n"))

	f.Fuzz(func(_ *testing.T, input []byte) {
		// Must not panic on any input
		parseSSEResponse(input)
	})
}

// FuzzValidateDataPrimeQuery tests query validation with arbitrary input
// to ensure the security-critical validation logic never panics.
func FuzzValidateDataPrimeQuery(f *testing.F) {
	f.Add("source logs | filter $l.applicationname == 'test'")
	f.Add("source logs | limit 100")
	f.Add("'; DROP TABLE users; --")
	f.Add("source logs | filter $d.field ~~ 'pattern'")
	f.Add("")
	f.Add("source logs | filter $l.severity >= 4 | orderby $l.timestamp desc")
	f.Add("UNION SELECT * FROM information_schema.tables")

	f.Fuzz(func(_ *testing.T, query string) {
		// Must not panic on any input
		_ = ValidateDataPrimeQuery(query)
	})
}

// FuzzValidateNoInjectionPatterns tests injection detection with arbitrary
// input to ensure the security boundary never panics.
func FuzzValidateNoInjectionPatterns(f *testing.F) {
	f.Add("SELECT * FROM users")
	f.Add("source logs | limit 10")
	f.Add("'; DROP TABLE --")
	f.Add("UNION ALL SELECT password FROM credentials")
	f.Add("/* comment */ source logs")

	f.Fuzz(func(_ *testing.T, query string) {
		// Must not panic on any input
		_ = validateNoInjectionPatterns(query)
	})
}

// FuzzCursorRoundtrip tests cursor encoding/decoding with arbitrary input
// to ensure pagination cursor handling never panics and roundtrips correctly.
func FuzzCursorRoundtrip(f *testing.F) {
	f.Add("2024-01-15T10:00:00Z", "abc-123", "forward", 0, 10)
	f.Add("", "", "backward", 100, 50)
	f.Add("2024-06-01T00:00:00Z", "", "forward", 50, 25)

	f.Fuzz(func(t *testing.T, timestamp, lastID, direction string, offset, limit int) {
		// Skip invalid UTF-8 inputs — JSON marshal replaces invalid bytes
		// with U+FFFD, so roundtrip equality can't hold for non-UTF-8 strings.
		if !utf8.ValidString(timestamp) || !utf8.ValidString(lastID) || !utf8.ValidString(direction) {
			return
		}

		cursor := &PaginationCursor{
			Type:      CursorTypeTime,
			Timestamp: timestamp,
			LastID:    lastID,
			Direction: direction,
			Offset:    offset,
			Limit:     limit,
		}

		// Encode must not panic
		encoded := EncodeCursor(cursor)

		// Decode must not panic
		decoded, err := DecodeCursor(encoded)
		if err != nil {
			return // Invalid cursor is fine, just don't panic
		}

		// If decode succeeded, verify roundtrip fidelity
		if decoded.Timestamp != cursor.Timestamp {
			t.Errorf("Timestamp mismatch: encoded %q, decoded %q", cursor.Timestamp, decoded.Timestamp)
		}
		if decoded.LastID != cursor.LastID {
			t.Errorf("LastID mismatch: encoded %q, decoded %q", cursor.LastID, decoded.LastID)
		}
		if decoded.Direction != cursor.Direction {
			t.Errorf("Direction mismatch: encoded %q, decoded %q", cursor.Direction, decoded.Direction)
		}
		if decoded.Offset != cursor.Offset {
			t.Errorf("Offset mismatch: encoded %d, decoded %d", cursor.Offset, decoded.Offset)
		}
	})
}

// FuzzDecodeCursor tests cursor decoding with arbitrary base64 input
// to ensure we never panic on malformed cursor strings.
func FuzzDecodeCursor(f *testing.F) {
	f.Add("")
	f.Add("not-base64")
	f.Add("eyJ0eXBlIjoidGltZSIsInZhbHVlIjoiMjAyNCJ9") // valid base64, valid JSON  pragma: allowlist secret
	f.Add("dGVzdA==")                                 // valid base64, not JSON
	f.Add("e30=")                                     // valid base64, empty JSON object

	f.Fuzz(func(_ *testing.T, input string) {
		// Must not panic on any input
		_, _ = DecodeCursor(input)
	})
}

// FuzzSuggestQueryFix tests query fix suggestions with arbitrary error messages
// to ensure the suggestion engine never panics.
func FuzzSuggestQueryFix(f *testing.F) {
	f.Add("invalid syntax near 'source'")
	f.Add("unknown field 'foo'")
	f.Add("")
	f.Add("unexpected token at position 42")
	f.Add("field type mismatch: expected string, got number")

	f.Fuzz(func(_ *testing.T, errorMessage string) {
		// Must not panic on any input
		_ = SuggestQueryFix("source logs | limit 10", errorMessage)
	})
}

// FuzzFormatQueryError tests error formatting with arbitrary inputs
// to ensure the formatter never panics.
func FuzzFormatQueryError(f *testing.F) {
	f.Add("source logs", "syntax error")
	f.Add("", "")
	f.Add("very long query "+string(make([]byte, 1000)), "error")
	f.Add("source logs | filter a == 'b'", "unexpected token '|' at position 13")

	f.Fuzz(func(_ *testing.T, query, apiError string) {
		// Must not panic on any input
		_ = FormatQueryError(query, apiError)
	})
}
