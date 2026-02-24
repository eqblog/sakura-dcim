package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/sakura-dcim/sakura-dcim/backend/internal/domain"
)

func TestTenantTree_Build(t *testing.T) {
	rootID := uuid.New()
	childID := uuid.New()
	grandchildID := uuid.New()

	flat := []domain.Tenant{
		{ID: rootID, Name: "Root"},
		{ID: childID, ParentID: &rootID, Name: "Child"},
		{ID: grandchildID, ParentID: &childID, Name: "Grandchild"},
	}

	root := BuildTree(flat, rootID)

	if root == nil {
		t.Fatal("root not found")
	}
	if root.Name != "Root" {
		t.Errorf("expected root name 'Root', got %s", root.Name)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(root.Children))
	}
	if root.Children[0].Name != "Child" {
		t.Errorf("expected child name 'Child', got %s", root.Children[0].Name)
	}
	if len(root.Children[0].Children) != 1 {
		t.Fatalf("expected 1 grandchild, got %d", len(root.Children[0].Children))
	}
	if root.Children[0].Children[0].Name != "Grandchild" {
		t.Errorf("expected grandchild name 'Grandchild', got %s", root.Children[0].Children[0].Name)
	}
}

func TestBuildTree_UnknownRoot(t *testing.T) {
	flat := []domain.Tenant{
		{ID: uuid.New(), Name: "Orphan"},
	}
	result := BuildTree(flat, uuid.New())
	if result != nil {
		t.Error("expected nil for unknown root ID")
	}
}

func TestTenantCreateRequest_Validation(t *testing.T) {
	req := TenantCreateRequest{
		Name: "Test Tenant",
		Slug: "test-tenant",
	}
	if req.Name == "" {
		t.Error("name should not be empty")
	}
	if req.Slug == "" {
		t.Error("slug should not be empty")
	}
}
