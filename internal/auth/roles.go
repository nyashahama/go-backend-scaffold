package auth

// Role represents a user's role within an org.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func IsOwnerRole(role string) bool {
	return role == string(RoleOwner)
}

func IsAdminRole(role string) bool {
	return role == string(RoleAdmin)
}

func IsMemberRole(role string) bool {
	return role == string(RoleMember)
}
