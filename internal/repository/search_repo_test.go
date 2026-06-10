package repository_test

import (
	"encoding/json"
	"testing"

	"github.com/merkulovlad/nuhach/internal/repository"
)

func TestBuildSearchQuery(t *testing.T) {
	query, err := repository.BuildSearchQuery("роза", 10, 0)
	if err != nil {
		t.Fatalf("BuildSearchQuery() error = %v", err)
	}

	// Parse the JSON to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(query, &result); err != nil {
		t.Fatalf("Failed to unmarshal query: %v", err)
	}

	// Verify query structure exists
	queryObj, ok := result["query"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing 'query' in search query")
	}

	// Verify function_score exists
	funcScore, ok := queryObj["function_score"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing 'function_score' in query")
	}

	// Verify inner query exists
	innerQuery, ok := funcScore["query"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing inner 'query' in function_score")
	}

	// Verify multi_match exists
	multiMatch, ok := innerQuery["multi_match"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing 'multi_match' in inner query")
	}

	// Verify query text
	if multiMatch["query"] != "роза" {
		t.Errorf("Query text = %v, want 'роза'", multiMatch["query"])
	}

	// Verify type is best_fields
	if multiMatch["type"] != "best_fields" {
		t.Errorf("Query type = %v, want 'best_fields'", multiMatch["type"])
	}

	// Verify fuzziness
	if multiMatch["fuzziness"] != "AUTO" {
		t.Errorf("Fuzziness = %v, want 'AUTO'", multiMatch["fuzziness"])
	}

	// Verify fields are present with boosts
	fields, ok := multiMatch["fields"].([]interface{})
	if !ok {
		t.Fatal("Missing 'fields' in multi_match")
	}

	expectedFields := map[string]bool{
		"name^5":           false,
		"brand_en^4":       false,
		"accords_ru^3":     false,
		"accords_en^2.5":   false,
		"notes_ru^2":       false,
		"notes_en^1.5":     false,
		"perfumers_en^1.2": false,
	}

	for _, f := range fields {
		if fieldStr, ok := f.(string); ok {
			expectedFields[fieldStr] = true
		}
	}

	for field, found := range expectedFields {
		if !found {
			t.Errorf("Missing field %s in search query", field)
		}
	}

	// Verify functions array exists
	functions, ok := funcScore["functions"].([]interface{})
	if !ok {
		t.Fatal("Missing 'functions' in function_score")
	}

	if len(functions) < 2 {
		t.Errorf("Expected at least 2 functions, got %d", len(functions))
	}

	// Verify pagination
	if result["from"] != float64(0) {
		t.Errorf("'from' = %v, want 0", result["from"])
	}

	if result["size"] != float64(10) {
		t.Errorf("'size' = %v, want 10", result["size"])
	}
}

func TestBuildSearchQuery_WithOffset(t *testing.T) {
	query, err := repository.BuildSearchQuery("test", 20, 40)
	if err != nil {
		t.Fatalf("BuildSearchQuery() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(query, &result); err != nil {
		t.Fatalf("Failed to unmarshal query: %v", err)
	}

	if result["from"] != float64(40) {
		t.Errorf("'from' = %v, want 40", result["from"])
	}

	if result["size"] != float64(20) {
		t.Errorf("'size' = %v, want 20", result["size"])
	}
}

func TestSearchQueryFieldCount(t *testing.T) {
	query, _ := repository.BuildSearchQuery("test", 10, 0)

	var result map[string]interface{}
	if err := json.Unmarshal(query, &result); err != nil {
		t.Fatalf("Failed to unmarshal query: %v", err)
	}

	queryObj := result["query"].(map[string]interface{})
	funcScore := queryObj["function_score"].(map[string]interface{})
	innerQuery := funcScore["query"].(map[string]interface{})
	multiMatch := innerQuery["multi_match"].(map[string]interface{})
	fields := multiMatch["fields"].([]interface{})

	const want = 7
	if len(fields) != want {
		t.Errorf("Expected %d fields with boosts, got %d", want, len(fields))
	}
}
