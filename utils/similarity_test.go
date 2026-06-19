package utils_test

import (
	"math"
	"testing"

	"my-app/utils"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		v1       []float32
		v2       []float32
		expected float32
	}{
		{
			name:     "Identical vectors",
			v1:       []float32{1.0, 0.0, 0.0},
			v2:       []float32{1.0, 0.0, 0.0},
			expected: 1.0,
		},
		{
			name:     "Orthogonal vectors",
			v1:       []float32{1.0, 0.0, 0.0},
			v2:       []float32{0.0, 1.0, 0.0},
			expected: 0.0,
		},
		{
			name:     "45-degree angle vectors",
			v1:       []float32{1.0, 0.0, 0.0},
			v2:       []float32{1.0, 1.0, 0.0}, // Cosine of 45 deg = 1 / sqrt(2) ≈ 0.7071
			expected: float32(1.0 / math.Sqrt(2.0)),
		},
		{
			name:     "Opposite direction vectors",
			v1:       []float32{0.0, 2.0},
			v2:       []float32{0.0, -2.0},
			expected: -1.0,
		},
		{
			name:     "Empty vectors",
			v1:       []float32{},
			v2:       []float32{},
			expected: 0.0,
		},
		{
			name:     "Mismatched dimensions",
			v1:       []float32{1.0, 2.0},
			v2:       []float32{1.0, 2.0, 3.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.CosineSimilarity(tt.v1, tt.v2)
			
			// Allow a small delta for floating point precision
			diff := math.Abs(float64(got - tt.expected))
			if diff > 1e-6 {
				t.Errorf("CosineSimilarity() = %v, expected %v (diff %v)", got, tt.expected, diff)
			}
		})
	}
}
