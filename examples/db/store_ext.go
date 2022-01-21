package db

import (
	"context"

	"github.com/semrekkers/endo/pkg/endo"
)

// This file extends the generated code in store.go and reuses some of the queries.

// ExpandRolesInUser expands the roles in the user.
func (s *Store) ExpandRolesInUser(ctx context.Context, u *User) error {
	// The query reuses the querySelectRole from the generated code.
	const query = querySelectRole + "JOIN user_roles ON roles.id = user_roles.role_id WHERE user_roles.user_id = $1"

	return s.tx(ctx, endo.TxReadOnly, func(dbtx endo.DBTX) error {
		rows, err := dbtx.QueryContext(ctx, query, u.ID)
		if err != nil {
			return err
		}
		defer rows.Close()
		// When using the querySelectX, you can use scanXRows to scan the rows.
		u.Roles, err = scanRoleRows(rows)
		return err
	})
}
