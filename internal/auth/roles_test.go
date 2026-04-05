package auth

import "testing"

func TestIsAdminRole(t *testing.T) {
	if !IsAdminRole("admin") {
		t.Fatal("expected admin to be recognized as admin role")
	}
	if IsAdminRole("member") {
		t.Fatal("did not expect member to be recognized as admin role")
	}
	if IsAdminRole("") {
		t.Fatal("did not expect empty string to be recognized as admin role")
	}
}

func TestIsMemberRole(t *testing.T) {
	if !IsMemberRole("member") {
		t.Fatal("expected member to be recognized as member role")
	}
	if IsMemberRole("admin") {
		t.Fatal("did not expect admin to be recognized as member role")
	}
	if IsMemberRole("") {
		t.Fatal("did not expect empty string to be recognized as member role")
	}
}
