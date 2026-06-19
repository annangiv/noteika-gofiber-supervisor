package utils

import (
	"strings"
	"unicode"
)

// NormalizeText lowercases, trims, and collapses whitespace for text comparison.
func NormalizeText(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(unicode.ToLower(r))
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}

// TextSimilarity returns a 0–1 score for how alike two text blobs are.
// Uses exact match, substring overlap, and word Jaccard — catches copy-paste
// duplicates even when embeddings are missing or stale.
func TextSimilarity(a, b string) float32 {
	a = NormalizeText(a)
	b = NormalizeText(b)
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}

	// One string fully contained in the other
	if strings.Contains(a, b) || strings.Contains(b, a) {
		shorter, longer := len(a), len(b)
		if shorter > longer {
			shorter, longer = longer, shorter
		}
		ratio := float32(shorter) / float32(longer)
		if ratio > 0.85 {
			return ratio
		}
	}

	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}

	setA := make(map[string]struct{}, len(wordsA))
	for _, w := range wordsA {
		setA[w] = struct{}{}
	}

	intersection := 0
	setB := make(map[string]struct{}, len(wordsB))
	for _, w := range wordsB {
		if _, seen := setB[w]; seen {
			continue
		}
		setB[w] = struct{}{}
		if _, ok := setA[w]; ok {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}
	return float32(intersection) / float32(union)
}

// CombinedSimilarity returns the best of vector and text similarity.
func CombinedSimilarity(query string, captureTitle, captureBody string, captureTags []string, queryVector, captureVector []float32) float32 {
	captureText := CaptureSearchText(captureTitle, captureBody, captureTags)
	textScore := TextSimilarity(query, captureText)

	var vectorScore float32
	if len(queryVector) > 0 && len(captureVector) > 0 {
		vectorScore = CosineSimilarity(queryVector, captureVector)
	}

	if textScore > vectorScore {
		return textScore
	}
	return vectorScore
}
