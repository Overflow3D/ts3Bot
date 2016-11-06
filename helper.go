package main

import (
	"strings"
)

func escape(sIn string) string {
	r := strings.NewReplacer(
		"\\", "\\\\",
		" ", "\\s",
		"|", "\\p",
		"\a", "\\a",
		"\b", "\\b",
		"\f", "\\f",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
		"\v", "\\v",
	)

	return r.Replace(sIn)
}
func unescape(sIn string) string {
	r := strings.NewReplacer(
		"\\v", "\v",
		"\\t", "\t",
		"\\r", "\r",
		"\\n", "\n",
		"\\f", "\f",
		"\\b", "\b",
		"\\a", "\a",
		"\\p", "|",
		"\\s", " ",
		"\\\\", "\\",
	)

	return r.Replace(sIn)
}
