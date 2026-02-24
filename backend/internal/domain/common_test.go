package domain

import (
	"encoding/json"
	"testing"
)

func TestAPIResponse_JSON(t *testing.T) {
	resp := APIResponse{Success: true, Data: "hello", Message: "ok"}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded APIResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if !decoded.Success {
		t.Error("expected success=true")
	}
	if decoded.Message != "ok" {
		t.Errorf("expected message=ok, got %s", decoded.Message)
	}
}

func TestAPIResponse_Error(t *testing.T) {
	resp := APIResponse{Success: false, Error: "not found"}
	data, _ := json.Marshal(resp)

	var decoded APIResponse
	json.Unmarshal(data, &decoded)

	if decoded.Success {
		t.Error("expected success=false")
	}
	if decoded.Error != "not found" {
		t.Errorf("expected error='not found', got %s", decoded.Error)
	}
}

func TestPaginatedResult(t *testing.T) {
	result := PaginatedResult[string]{
		Items:      []string{"a", "b", "c"},
		Total:      10,
		Page:       1,
		PageSize:   3,
		TotalPages: 4,
	}

	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}
	if result.Total != 10 {
		t.Errorf("expected total=10, got %d", result.Total)
	}
	if result.TotalPages != 4 {
		t.Errorf("expected 4 total pages, got %d", result.TotalPages)
	}
}
