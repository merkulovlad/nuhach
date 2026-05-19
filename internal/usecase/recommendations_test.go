package usecase_test

import (
	"math"
	"testing"

	"github.com/merkulovlad/nuhach/internal/domain"
	"github.com/merkulovlad/nuhach/internal/usecase"
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
	// User vector points along +X; new item points along +Y with positive
	// weight. Centroid should rotate from (1,0) toward +Y and stay unit.
	user := []float32{1, 0}
	item := []float32{0, 1}

	got := usecase.MergeEmbedding(user, item, 1.0, 0.95)

	if got[1] <= 0 {
		t.Errorf("expected positive y-component after positive-weight merge, got %v", got)
	}
	if got[0] <= 0 {
		t.Errorf("expected positive x-component preserved, got %v", got)
	}
	norm := math.Sqrt(float64(got[0]*got[0] + got[1]*got[1]))
	if math.Abs(norm-1.0) > 1e-5 {
		t.Errorf("result not normalized, |v|=%v", norm)
	}
}

func TestMergeEmbedding_NegativeWeightPushesAway(t *testing.T) {
	// Disliked item along +Y must push the y-component negative.
	user := []float32{1, 0}
	item := []float32{0, 1}

	got := usecase.MergeEmbedding(user, item, -0.4, 0.95)

	if got[1] >= 0 {
		t.Errorf("expected y-component to go negative after dislike, got %v", got)
	}
}

func TestMergeEmbedding_NoNaNOnZeroResult(t *testing.T) {
	// Dislike of the same vector the user already has could in principle
	// produce a zero vector; normalize must not divide by zero.
	user := []float32{1, 0, 0}
	item := []float32{1, 0, 0}

	got := usecase.MergeEmbedding(user, item, -1.0, 0.5)

	for i, v := range got {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			t.Fatalf("non-finite value at index %d: %v", i, v)
		}
	}
}

func TestMergeEmbedding_HighDecayMovesLess(t *testing.T) {
	// With decay near 1, a single event moves the centroid only a tiny
	// bit. With low decay, the centroid follows the new item aggressively.
	user := []float32{1, 0, 0}
	item := []float32{0, 1, 0}

	slow := usecase.MergeEmbedding(user, item, 1.0, 0.99)
	fast := usecase.MergeEmbedding(user, item, 1.0, 0.5)

	simSlow := usecase.CosineSimilarity(user, slow)
	simFast := usecase.CosineSimilarity(user, fast)

	if simSlow <= simFast {
		t.Errorf("expected high-decay centroid to stay closer to original (slow=%v, fast=%v)", simSlow, simFast)
	}
	if simSlow < 0.99 {
		t.Errorf("decay=0.99 moved centroid too much: cos=%v", simSlow)
	}
}

func TestMergeEmbedding_UsesDecayEMAFormula(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
		decay  float64
		want   []float32
	}{
		{
			name:   "positive weight",
			weight: 1.0,
			decay:  0.8,
			want:   []float32{0.9701425, 0.2425356},
		},
		{
			name:   "negative weight",
			weight: -0.5,
			decay:  0.8,
			want:   []float32{0.9922779, -0.1240347},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := []float32{1, 0}
			item := []float32{0, 1}

			got := usecase.MergeEmbedding(user, item, tt.weight, tt.decay)

			assertFloat32SliceApprox(t, got, tt.want, 1e-6)
		})
	}
}

func TestMergeEmbedding_DecayOneIgnoresPerfume(t *testing.T) {
	// decay=1 should keep only the normalized user history.
	user := []float32{3, 4}
	item := []float32{0, 100}

	got := usecase.MergeEmbedding(user, item, 1.0, 1.0)

	assertFloat32SliceApprox(t, got, []float32{0.6, 0.8}, 1e-6)
}

func TestMergeEmbedding_DoesNotMutateInputs(t *testing.T) {
	user := []float32{1, 0}
	item := []float32{0, 1}

	_ = usecase.MergeEmbedding(user, item, 1.0, 0.8)

	assertFloat32SliceApprox(t, user, []float32{1, 0}, 0)
	assertFloat32SliceApprox(t, item, []float32{0, 1}, 0)
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func assertFloat32SliceApprox(t *testing.T, got, want []float32, tolerance float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if math.Abs(float64(got[i]-want[i])) > tolerance {
			t.Fatalf("got[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}
