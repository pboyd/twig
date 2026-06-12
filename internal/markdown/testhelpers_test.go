package markdown

import "strings"

// stripAnsi removes ANSI/OSC escape sequences from s, returning bare text.
// This is a minimal implementation sufficient for test assertions.
func stripAnsi(s string) string {
	var out strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			// OSC sequences (e.g. OSC 8 hyperlinks) end with BEL (\x07)
			if b == '\x07' {
				inEsc = false
			} else if b == 'm' {
				// CSI SGR sequences end with 'm'
				inEsc = false
			}
			continue
		}
		out.WriteByte(b)
	}
	return out.String()
}
