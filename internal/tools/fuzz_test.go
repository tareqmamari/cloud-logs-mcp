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
	f.Add(`source logs | filter $d.message == 'trailing backslash\'`) // trailing backslash before closing quote

	f.Fuzz(func(t *testing.T, query string) {
		// Must not panic on any input
		err := ValidateDataPrimeQuery(query)
		// Semantic check: a validator that "rejects" a query but gives no
		// actionable message is as much a bug as failing to reject an unsafe
		// query at all - the caller (and the LLM using it) has no idea why.
		if err != nil && err.Message == "" {
			t.Errorf("ValidateDataPrimeQuery(%q) returned a non-nil error with an empty Message", query)
		}
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
	f.Add(`foo\`) // trailing backslash

	f.Fuzz(func(t *testing.T, query string) {
		// Must not panic on any input
		err := validateNoInjectionPatterns(query)
		// Semantic check: same rationale as FuzzValidateDataPrimeQuery above.
		if err != nil && err.Message == "" {
			t.Errorf("validateNoInjectionPatterns(%q) returned a non-nil error with an empty Message", query)
		}
	})
}

// FuzzEscapeDataPrimeString tests escapeDataPrimeString (the function
// responsible for making caller-supplied values safe to embed in a
// DataPrime string literal - see query_builder.go) with arbitrary input, to
// ensure it never panics and, critically, never lets an unescaped quote
// through. An unescaped quote in the output would let a value break out of
// the DataPrime string literal it's embedded in and inject query syntax.
func FuzzEscapeDataPrimeString(f *testing.F) {
	f.Add("hello world")
	f.Add("it's a test")
	f.Add(`foo\`) // trailing backslash: the exact regression case called out
	// in escapeDataPrimeString's doc comment - without escaping the trailing
	// backslash, it could "consume" the closing quote the caller appends.
	f.Add(`\'`)
	f.Add("line1\nline2")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		out := escapeDataPrimeString(input)

		// Semantic check: every quote in the escaped output must be
		// immediately preceded by the backslash escapeDataPrimeString adds
		// for it. escapeDataPrimeString only ever emits a quote as part of
		// the two-character `\'` sequence, so this holds by construction -
		// a failure here means a caller-controlled quote is escaping
		// unescaped.
		//
		// Checking only "is the immediately preceding rune a backslash"
		// has a blind spot: a run of backslashes immediately before the
		// quote pairs up two at a time (each `\\` is one escaped literal
		// backslash), so only an ODD-length run actually escapes the quote
		// - an even-length run (e.g. `\\\\'`, four backslashes then an
		// unescaped quote) would wrongly read as "escaped" under a
		// single-rune check. Count the full run instead.
		runes := []rune(out)
		for i, r := range runes {
			if r != '\'' {
				continue
			}
			backslashRun := 0
			for j := i - 1; j >= 0 && runes[j] == '\\'; j-- {
				backslashRun++
			}
			if backslashRun%2 == 0 {
				t.Errorf("escapeDataPrimeString(%q) produced an unescaped quote at index %d (preceded by an even run of %d backslashes) in output %q", input, i, backslashRun, out)
			}
		}
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
