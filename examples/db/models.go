package db

import (
	"database/sql"
	"time"
)

//go:generate endogen -out store.go

// User represents an application user.
type User struct {
	ID            int            `db:"id,primary"`
	Email         string         `db:"email"`
	FirstName     sql.NullString `db:"first_name"`
	LastName      sql.NullString `db:"last_name"`
	DisplayName   sql.NullString `db:"display_name,readonly"`
	EmailVerified bool           `db:"email_verified"`
	PasswordHash  sql.NullString `db:"password_hash"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`

	Roles []*Role `db:"-"`
}

// Role represents an application role.
type Role struct {
	ID   int    `db:"id,primary"`
	Name string `db:"name"`
}
