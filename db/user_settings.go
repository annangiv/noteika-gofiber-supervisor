package db

const (
	DefaultSearchMinSimilarity = float32(0.70)
	MinSearchMinSimilarity     = float32(0.50)
	MaxSearchMinSimilarity     = float32(0.85)
)

// EffectiveSearchMinSimilarity returns the user's preference clamped to allowed range,
// or the default when unset (0).
func EffectiveSearchMinSimilarity(value float32) float32 {
	if value <= 0 {
		return DefaultSearchMinSimilarity
	}
	if value < MinSearchMinSimilarity {
		return MinSearchMinSimilarity
	}
	if value > MaxSearchMinSimilarity {
		return MaxSearchMinSimilarity
	}
	return value
}
