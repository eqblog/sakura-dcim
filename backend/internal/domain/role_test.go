package domain

import "testing"

func TestRole_HasPermission(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		perm        string
		want        bool
	}{
		{"has exact permission", []string{"server.view", "server.edit"}, "server.view", true},
		{"missing permission", []string{"server.view"}, "server.delete", false},
		{"wildcard grants all", []string{"*"}, "anything", true},
		{"empty permissions", nil, "server.view", false},
		{"multiple check middle", []string{"a", "b", "c"}, "b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Role{Permissions: tt.permissions}
			if got := r.HasPermission(tt.perm); got != tt.want {
				t.Errorf("HasPermission(%q) = %v, want %v", tt.perm, got, tt.want)
			}
		})
	}
}

func TestAllPermissions_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range AllPermissions {
		if seen[p] {
			t.Errorf("duplicate permission: %s", p)
		}
		seen[p] = true
	}
}

func TestAllPermissions_Count(t *testing.T) {
	if len(AllPermissions) < 20 {
		t.Errorf("expected at least 20 permissions, got %d", len(AllPermissions))
	}
}
