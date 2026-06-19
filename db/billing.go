package db

import "os"

const (
	TierFree = "free"
	TierPro  = "pro"
)

const DefaultFreeCaptureLimit = 10

func FreeCaptureLimit() int {
	// Optional override via env; invalid values fall back to default.
	raw := os.Getenv("NOTEIKA_FREE_CAPTURE_LIMIT")
	if raw == "" {
		return DefaultFreeCaptureLimit
	}
	var n int
	for _, c := range raw {
		if c < '0' || c > '9' {
			return DefaultFreeCaptureLimit
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return DefaultFreeCaptureLimit
	}
	return n
}

func IsProTier(tier string) bool {
	return tier == TierPro
}
