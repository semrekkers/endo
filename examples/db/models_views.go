package db

//go:generate endogen -gen-store=false --views -out store_views.go

// EffectiveRole (table: effective_roles) represents an effective role for a user.
//
// order by: "user_id, role_id"
type EffectiveRole struct {
	UserID   int    `db:"user_id,primary,exclude"`
	RoleID   int    `db:"role_id"`
	RoleName string `db:"role_name"`
}
