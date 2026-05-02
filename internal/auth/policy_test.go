package auth

import "testing"

func TestCanAccessOrg(t *testing.T) {
	identity := Identity{
		UserID: "user-123",
		OrgID:  "org-123",
		Role:   string(RoleMember),
	}

	if !CanAccessOrg(identity, "org-123") {
		t.Fatal("expected identity to access its own org")
	}
	if CanAccessOrg(identity, "org-456") {
		t.Fatal("did not expect identity to access another org")
	}
	if CanAccessOrg(Identity{}, "org-123") {
		t.Fatal("did not expect empty identity to access an org")
	}
	if CanAccessOrg(identity, "") {
		t.Fatal("did not expect identity to access an empty org")
	}
}

func TestHasRole(t *testing.T) {
	identity := Identity{
		UserID: "user-123",
		OrgID:  "org-123",
		Role:   string(RoleAdmin),
	}

	if !HasRole(identity, RoleAdmin) {
		t.Fatal("expected admin identity to have admin role")
	}
	if HasRole(identity, RoleMember) {
		t.Fatal("did not expect admin identity to have member role")
	}
	if HasRole(Identity{}, RoleAdmin) {
		t.Fatal("did not expect empty identity to have admin role")
	}
}

func TestHasAnyRole(t *testing.T) {
	identity := Identity{
		UserID: "user-123",
		OrgID:  "org-123",
		Role:   string(RoleAdmin),
	}

	if !HasAnyRole(identity, RoleOwner, RoleAdmin) {
		t.Fatal("expected admin identity to match one allowed role")
	}
	if HasAnyRole(identity, RoleMember) {
		t.Fatal("did not expect admin identity to match member-only role")
	}
	if HasAnyRole(identity) {
		t.Fatal("did not expect empty allowed role list to match")
	}
}
