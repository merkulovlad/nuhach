package usecase_test

import (
	"math"
	"testing"

	"nuhach/internal/usecase"
)

func TestComputeWeightedRating_WithValidData(t *testing.T) {
	globalMean := 3.5
	bayesianM := 10.0

	tests := []struct {
		name        string
		ratingValue float64
		ratingCount int
		expected    float64
	}{
		{
			name:        "high rating many votes",
			ratingValue: 4.5,
			ratingCount: 100,
			expected:    4.409, // (100/(100+10))*4.5 + (10/(100+10))*3.5
		},
		{
			name:        "high rating few votes",
			ratingValue: 5.0,
			ratingCount: 5,
			expected:    4.0, // (5/(5+10))*5.0 + (10/(5+10))*3.5
		},
		{
			name:        "low rating many votes",
			ratingValue: 2.0,
			ratingCount: 50,
			expected:    2.25, // (50/(50+10))*2.0 + (10/(50+10))*3.5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := tt.ratingValue
			rc := tt.ratingCount
			result := usecase.ComputeWeightedRating(&rv, &rc, globalMean, bayesianM)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("ComputeWeightedRating() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComputeWeightedRating_WithMissingData(t *testing.T) {
	globalMean := 3.5
	bayesianM := 10.0

	tests := []struct {
		name        string
		ratingValue *float64
		ratingCount *int
		expected    float64
	}{
		{
			name:        "nil rating value",
			ratingValue: nil,
			ratingCount: intPtr(10),
			expected:    globalMean,
		},
		{
			name:        "nil rating count",
			ratingValue: float64Ptr(4.5),
			ratingCount: nil,
			expected:    globalMean,
		},
		{
			name:        "both nil",
			ratingValue: nil,
			ratingCount: nil,
			expected:    globalMean,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := usecase.ComputeWeightedRating(tt.ratingValue, tt.ratingCount, globalMean, bayesianM)
			if result != tt.expected {
				t.Errorf("ComputeWeightedRating() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        []float32{1, 1, 0},
			b:        []float32{1, 0, 0},
			expected: 0.707, // 1/sqrt(2)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := usecase.CosineSimilarity(tt.a, tt.b)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("CosineSimilarity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCosineSimilarity_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
	}{
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0.0,
		},
		{
			name:     "different lengths",
			a:        []float32{1, 2},
			b:        []float32{1, 2, 3},
			expected: 0.0,
		},
		{
			name:     "zero vectors",
			a:        []float32{0, 0, 0},
			b:        []float32{0, 0, 0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := usecase.CosineSimilarity(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("CosineSimilarity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNormalizeEmbedding(t *testing.T) {
	tests := []struct {
		name     string
		input    []float32
		expected []float32
	}{
		{
			name:     "unit vector",
			input:    []float32{1, 0, 0},
			expected: []float32{1, 0, 0},
		},
		{
			name:     "needs normalization",
			input:    []float32{3, 4, 0},
			expected: []float32{0.6, 0.8, 0}, // 3/5, 4/5
		},
		{
			name:     "all equal",
			input:    []float32{1, 1, 1},
			expected: []float32{0.577, 0.577, 0.577}, // 1/sqrt(3)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]float32, len(tt.input))
			copy(input, tt.input)
			usecase.NormalizeEmbedding(input)

			for i := range input {
				if math.Abs(float64(input[i])-float64(tt.expected[i])) > 0.01 {
					t.Errorf("NormalizeEmbedding()[%d] = %v, want %v", i, input[i], tt.expected[i])
				}
			}
		})
	}
}

func TestNormalizeEmbedding_ZeroVector(t *testing.T) {
	input := []float32{0, 0, 0}
	usecase.NormalizeEmbedding(input)

	// Zero vector should remain zero
	for i, v := range input {
		if v != 0 {
			t.Errorf("NormalizeEmbedding() on zero vector[%d] = %v, want 0", i, v)
		}
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}
