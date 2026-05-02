package exampledomain

// Resource is a placeholder domain record used to demonstrate package boundaries.
type Resource struct {
	ID    string `json:"id"`
	OrgID string `json:"org_id"`
	Name  string `json:"name"`
}
