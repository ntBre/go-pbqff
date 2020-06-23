package main

import "strings"

// CleanSplit splits a line using strings.Split and then removes
// empty entries
func CleanSplit(str, sep string) []string {
	lines := strings.Split(str, sep)
	clean := make([]string, 0, len(lines))
	for s := range lines {
		if lines[s] != "" {
			clean = append(clean, lines[s])
		}
	}
	return clean
}
