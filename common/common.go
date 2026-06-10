package common

import (
	"strings"
	"unicode"
)

func CleanString(s string) string {
	return strings.Map(func(r rune) rune {
		// Keep the rune if it is a letter, number, or standard punctuation
		if unicode.IsPrint(r) {
			return r
		}
		// Otherwise, drop it
		return -1
	}, s)
}
