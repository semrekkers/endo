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

// GetUsersFiltered gets all Users with filters applied from the database.
func (s *Store) GetUsersFiltered(ctx context.Context, po endo.PageOptions, filters ...endo.NamedArg) ([]*User, error) {
	var qb endo.Builder
	qb.Write(querySelectUser)
	if 0 < len(filters) {
		qb.Write("WHERE ").WriteNamedArgs("%s", " AND ", filters...).Write(" ")
	}
	limit, offset := po.Args()
	qb.Write("ORDER BY id ").
		WriteWithParams("LIMIT {} OFFSET {}", limit, offset)
	query, args := qb.Build()

	var c []*User
	err := s.tx(ctx, endo.TxReadOnly, func(dbtx endo.DBTX) error {
		rows, err := dbtx.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		c, err = scanUserRows(rows)
		return err
	})

	return c, err
}
