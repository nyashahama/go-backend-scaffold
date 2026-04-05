package auth

// Role represents a user's role within an org.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func IsAdminRole(role string) bool {
	return role == string(RoleAdmin)
}

func IsMemberRole(role string) bool {
	return role == string(RoleMember)
}
