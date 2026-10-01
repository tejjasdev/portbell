// Package safe makes untrusted text safe to print in a terminal.
//
// Request headers and bodies come from strangers. They can contain control
// characters (for example ANSI escape sequences) that change what a terminal
// shows or does. Text turns such characters into visible escapes like \x1b.
package safe

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Text returns s with control characters, invalid UTF-8 bytes and Unicode
// bidirectional override characters replaced by visible escapes. Newline and
// tab are kept.
func Text(s string) string {
	if isClean(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			fmt.Fprintf(&b, `\x%02x`, s[i])
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case unicode.IsControl(r):
			if r < 0x100 {
				fmt.Fprintf(&b, `\x%02x`, r)
			} else {
				fmt.Fprintf(&b, `\u%04x`, r)
			}
		case isBidi(r):
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
		i += size
	}
	return b.String()
}

// Line is like Text but also escapes newlines and tabs, for single-line fields.
func Line(s string) string {
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return Text(s)
}

func isClean(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r == '\n' || r == '\t' {
			continue
		}
		if unicode.IsControl(r) || isBidi(r) {
			return false
		}
	}
	return true
}

func isBidi(r rune) bool {
	switch {
	case r >= 0x202A && r <= 0x202E: // embeddings and overrides
		return true
	case r >= 0x2066 && r <= 0x2069: // isolates
		return true
	case r == 0x200E || r == 0x200F || r == 0x061C: // marks
		return true
	}
	return false
}
