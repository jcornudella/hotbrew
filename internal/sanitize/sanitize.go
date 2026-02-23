package sanitize

// Text strips control characters (except newline and tab) to avoid ANSI/OSC injection.
func Text(s string) string {
	if s == "" {
		return s
	}
	var out []rune
	for _, r := range s {
		if r == '\n' || r == '\t' {
			out = append(out, r)
			continue
		}
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
