package docker

import "strings"

func Sanitize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else if r == '_' || r == '/' {
			b.WriteRune('-')
		}
	}
	out := b.String()
	if out == "" {
		return "default"
	}
	return out
}
