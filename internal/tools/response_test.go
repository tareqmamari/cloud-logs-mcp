package tools

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTruncateUTF8Safe locks down the rune-boundary-safe truncation helper
// used everywhere response.go previously did a raw byte-index slice
// (text[:limit], jsonBytes[:limit]). A raw slice at an arbitrary byte offset
// can land in the middle of a multi-byte UTF-8 rune, producing invalid UTF-8
// (and, for the JSON case, a corrupted/unparseable JSON document).
func TestTruncateUTF8Safe(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "ascii within limit is unchanged",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "ascii truncated cleanly",
			input:  "hello world",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "zero maxLen returns empty",
			input:  "hello",
			maxLen: 0,
			want:   "",
		},
		{
			name: "2-byte rune split at the boundary backs off before it",
			// "café" = c,a,f (1 byte each) + é (2 bytes: 0xC3 0xA9) = 5 bytes.
			// maxLen=4 would slice right through é's 2 bytes.
			input:  "café",
			maxLen: 4,
			want:   "caf",
		},
		{
			name:   "2-byte rune that fits exactly is kept whole",
			input:  "café",
			maxLen: 5,
			want:   "café",
		},
		{
			name: "4-byte emoji rune split at the boundary backs off before it",
			// "hi" + 😀 (4 bytes: F0 9F 98 80) + "bye". maxLen=4 lands inside the emoji.
			input:  "hi\U0001F600bye",
			maxLen: 4,
			want:   "hi",
		},
		{
			name:   "4-byte emoji rune that fits exactly is kept whole",
			input:  "hi\U0001F600bye",
			maxLen: 6,
			want:   "hi\U0001F600",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateUTF8Safe(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateUTF8Safe(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("truncateUTF8Safe(%q, %d) = %q is not valid UTF-8", tt.input, tt.maxLen, got)
			}
			if len(got) > tt.maxLen && tt.maxLen > 0 {
				t.Errorf("truncateUTF8Safe(%q, %d) = %q (%d bytes) exceeds maxLen", tt.input, tt.maxLen, got, len(got))
			}
		})
	}
}

// TestTruncateBytesUTF8Safe mirrors TestTruncateUTF8Safe for the []byte
// variant used on jsonBytes.
func TestTruncateBytesUTF8Safe(t *testing.T) {
	input := []byte("café" + strings.Repeat("x", 10))
	// 5 bytes for "café", truncate to 4 to land mid-rune.
	got := truncateBytesUTF8Safe(input, 4)
	if string(got) != "caf" {
		t.Errorf("truncateBytesUTF8Safe = %q, want %q", got, "caf")
	}
	if !utf8.Valid(got) {
		t.Errorf("truncateBytesUTF8Safe result is not valid UTF-8: %q", got)
	}
}

// TestEnsureResponseLimit_MultiByteBoundary is an end-to-end regression test:
// a response whose FinalResponseLimit-TruncationBufferSize cut point lands
// mid-rune must still produce valid UTF-8 output.
func TestEnsureResponseLimit_MultiByteBoundary(t *testing.T) {
	cut := FinalResponseLimit - TruncationBufferSize
	// Build a string where a multi-byte rune straddles the cut point exactly.
	prefix := strings.Repeat("a", cut-1)
	text := prefix + "é" + strings.Repeat("b", 100) // 'é' (2 bytes) starts one byte before the cut

	got := ensureResponseLimit(text, nil)
	if !utf8.ValidString(got) {
		t.Fatalf("ensureResponseLimit produced invalid UTF-8 at a multi-byte boundary")
	}
}
