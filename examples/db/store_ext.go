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

	return s.TX(ctx, endo.TxReadOnly, func(dbtx endo.DBTX) error {
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

// GetExpandedUsers returns fully expanded User entities, filters can optionally be applied.
func (s *Store) GetExpandedUsers(ctx context.Context, po endo.PageOptions, filters ...endo.NamedArg) ([]*User, error) {
	var c []*User

	// Use existing Store methods inside one transaction, note the endo.TxMulti flag.
	err := s.TX(ctx, endo.TxMulti|endo.TxReadOnly, func(dbtx endo.DBTX) error {
		var err error

		// Create the transactional Store (txs).
		txs := Store{endo.WrapTX(dbtx)}

		c, err = txs.GetUsersFiltered(ctx, po, filters...)
		if err != nil {
			return err
		}
		for _, user := range c {
			// Expand roles of every User in collection.
			if err = txs.ExpandRolesInUser(ctx, user); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}
