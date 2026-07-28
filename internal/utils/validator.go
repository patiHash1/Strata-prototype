package utils

import "regexp"

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// IsEmail returns true if the string looks like a valid email address.
func IsEmail(s string) bool {
	return emailRegex.MatchString(s)
}

// IsDomainSlug returns true if the string is a valid domain slug
// (lowercase alphanumeric + hyphens, 1–100 chars).
func IsDomainSlug(s string) bool {
	if len(s) == 0 || len(s) > 100 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
}

// NotBlank returns true if the trimmed string is non-empty.
func NotBlank(s string) bool {
	return len(s) > 0
}

// MinLen returns true if the string has at least n characters.
func MinLen(s string, n int) bool {
	return len(s) >= n
}
