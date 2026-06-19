package utils

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var hashtagPattern = regexp.MustCompile(`#([a-zA-Z][a-zA-Z0-9_-]{1,31})`)

// NormalizeTag lowercases and trims a tag for storage.
func NormalizeTag(tag string) string {
	tag = strings.TrimSpace(strings.ToLower(tag))
	tag = strings.TrimPrefix(tag, "#")
	tag = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			return r
		}
		if r == ' ' {
			return '-'
		}
		return -1
	}, tag)
	return strings.Trim(tag, "-_")
}

// ParseHashtags extracts #tags embedded in note body text.
func ParseHashtags(body string) []string {
	seen := make(map[string]struct{})
	var tags []string
	for _, match := range hashtagPattern.FindAllStringSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		normalized := NormalizeTag(match[1])
		if normalized == "" || len(normalized) < 2 {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		tags = append(tags, normalized)
	}
	sort.Strings(tags)
	return tags
}

// NormalizeTags cleans a list of user or suggested tags.
func NormalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{})
	var out []string
	for _, t := range tags {
		n := NormalizeTag(t)
		if n == "" || len(n) < 2 {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// MergeTags combines tag lists, deduplicating and sorting.
func MergeTags(lists ...[]string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, list := range lists {
		for _, t := range list {
			n := NormalizeTag(t)
			if n == "" || len(n) < 2 {
				continue
			}
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			out = append(out, n)
		}
	}
	sort.Strings(out)
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

// CaptureSearchText builds the text blob used for similarity scoring.
func CaptureSearchText(title, body string, tags []string) string {
	var parts []string
	if len(tags) > 0 {
		parts = append(parts, strings.Join(tags, " "))
	}
	if strings.TrimSpace(title) != "" {
		parts = append(parts, title)
	}
	if strings.TrimSpace(body) != "" {
		parts = append(parts, body)
	}
	return strings.Join(parts, "\n")
}
