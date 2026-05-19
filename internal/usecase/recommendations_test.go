package usecase_test

import (
	"math"
	"testing"

	"nuhach/internal/domain"
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

func TestEventEmbeddingWeight(t *testing.T) {
	tests := []struct {
		name   string
		event  domain.EventType
		rating *int
		want   float64
	}{
		{"like without rating", domain.EventLike, nil, 1.0},
		{"like with rating 5", domain.EventLike, intPtr(5), 1.0},
		{"like with rating 3", domain.EventLike, intPtr(3), 0.6},
		{"save", domain.EventSave, nil, 1.0},
		{"click", domain.EventClick, nil, 0.15},
		{"dislike pushes away", domain.EventDislike, nil, -0.4},
		{"impression is neutral", domain.EventImpression, nil, 0},
		{"my_saves is neutral", domain.EventMySaves, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := usecase.EventEmbeddingWeight(tt.event, tt.rating)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("EventEmbeddingWeight(%s) = %v, want %v", tt.event, got, tt.want)
			}
		})
	}
}

func TestMergeEmbedding_PositiveWeightPullsToward(t *testing.T) {
	// User vector points along +X; new item points along +Y with weight 1.
	// Centroid should rotate from (1,0) toward (1,1)/sqrt(2).
	user := []float32{1, 0}
	item := []float32{0, 1}

	got := usecase.MergeEmbedding(user, 1, item, 1.0)

	if got[1] <= 0 {
		t.Errorf("expected positive y-component after positive-weight merge, got %v", got)
	}
	if got[0] <= 0 {
		t.Errorf("expected positive x-component preserved, got %v", got)
	}
	// Result must be unit length.
	norm := math.Sqrt(float64(got[0]*got[0] + got[1]*got[1]))
	if math.Abs(norm-1.0) > 1e-5 {
		t.Errorf("result not normalized, |v|=%v", norm)
	}
}

func TestMergeEmbedding_NegativeWeightPushesAway(t *testing.T) {
	// User along +X. Disliked item also along +Y. After dislike the
	// centroid's y-component must be negative (moved away from +Y).
	user := []float32{1, 0}
	item := []float32{0, 1}

	got := usecase.MergeEmbedding(user, 1, item, -0.4)

	if got[1] >= 0 {
		t.Errorf("expected y-component to go negative after dislike, got %v", got)
	}
}

func TestMergeEmbedding_DenominatorPositiveOnDislike(t *testing.T) {
	// With n=0 and a negative weight, naive denom (n+weight) would be
	// negative and the merge would explode. Guard: we divide by n+|w|.
	user := []float32{1, 0, 0}
	item := []float32{0, 1, 0}

	got := usecase.MergeEmbedding(user, 0, item, -0.4)

	for i, v := range got {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Fatalf("non-finite value at index %d: %v", i, v)
		}
	}
}

func TestMergeEmbedding_LargeNDampensSingleEvent(t *testing.T) {
	// With many prior interactions, a single new event should barely
	// move the centroid. Cosine to the original user vector must stay
	// close to 1.
	user := []float32{1, 0, 0}
	item := []float32{0, 1, 0}

	got := usecase.MergeEmbedding(user, 1000, item, 1.0)

	sim := usecase.CosineSimilarity(user, got)
	if sim < 0.999 {
		t.Errorf("single event moved centroid too much with n=1000: cos=%v", sim)
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}
