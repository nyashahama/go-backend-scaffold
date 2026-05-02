package auth

// CanAccessOrg reports whether the authenticated identity belongs to orgID.
func CanAccessOrg(identity Identity, orgID string) bool {
	return identity.UserID != "" && identity.OrgID != "" && orgID != "" && identity.OrgID == orgID
}

// HasRole reports whether the authenticated identity has the required role.
func HasRole(identity Identity, role Role) bool {
	return identity.UserID != "" && identity.OrgID != "" && identity.Role == string(role)
}

// HasAnyRole reports whether the authenticated identity has one of the allowed roles.
func HasAnyRole(identity Identity, roles ...Role) bool {
	for _, role := range roles {
		if HasRole(identity, role) {
			return true
		}
	}
	return false
}
