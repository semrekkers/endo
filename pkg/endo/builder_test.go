package endo_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/semrekkers/endo/pkg/endo"

	"github.com/stretchr/testify/assert"
)

func TestWrite(t *testing.T) {
	var b endo.Builder

	query := b.Write("SELECT * FROM users").String()

	assert.Equal(t, "SELECT * FROM users", query)
}

func TestWriteTrim(t *testing.T) {
	var b endo.Builder

	query := b.
		WriteTrim(`
			SELECT * FROM users
		`).
		Write(" WHERE id IS NULL").
		String()

	assert.Equal(t, "SELECT * FROM users WHERE id IS NULL", query)
}

func TestEmptyArgs(t *testing.T) {
	var b endo.Builder

	_, args := b.
		WriteWithArgs("SELECT * FROM users").
		Build()

	if args != nil {
		t.Error("args must be nil")
	}
}

func TestSQLSelect(t *testing.T) {
	var b endo.Builder

	query, args := b.WriteTrim(`
			SELECT id, username, email FROM users
		`).
		WriteWithPlaced(" WHERE id = ?", 10).
		Build()

	assert.Equal(t, "SELECT id, username, email FROM users WHERE id = $1", query)
	assert.Equal(t, []interface{}{10}, args)
}

func TestSQLInsert(t *testing.T) {
	var b endo.Builder

	query, args := b.WriteTrim(`
			INSERT INTO users (username, email) VALUES ($2, $3)
		`).
		WithArgs(
			"admin",
			"user@example.com",
		).
		Build()

	assert.Equal(t, "INSERT INTO users (username, email) VALUES ($2, $3)", query)
	assert.Equal(t, []interface{}{"admin", "user@example.com"}, args)
}

func TestDynamicUpdate(t *testing.T) {
	var b endo.Builder
	patch := map[string]interface{}{
		"email": "test@example.com",
	}

	b.WriteWithArgs("UPDATE users SET updated_at = $1", time.Date(2021, 12, 22, 20, 58, 0, 0, time.UTC))
	for k, v := range patch {
		b.Write(", ")
		b.Write(k)
		b.WriteWithPlaced(" = ?", v)
	}
	b.WriteWithPlaced(" WHERE id = ?", 3)
	query, args := b.Build()

	assert.Equal(t, "UPDATE users SET updated_at = $1, email = $2 WHERE id = $3", query)
	assert.Equal(t, []interface{}{time.Date(2021, time.December, 22, 20, 58, 0, 0, time.UTC), "test@example.com", 3}, args)
}

func TestDynamicUpdate2(t *testing.T) {
	var b endo.Builder
	patch := map[string]interface{}{
		"email": "test@example.com",
	}

	b.WriteWithArgs("UPDATE users SET updated_at = $1", time.Date(2021, 12, 22, 20, 58, 0, 0, time.UTC))
	for k, v := range patch {
		b.WriteWithPlaced(fmt.Sprintf(", %s = ?", k), v)
	}
	b.WriteWithPlaced(" WHERE id = ?", 3)
	query, args := b.Build()

	assert.Equal(t, "UPDATE users SET updated_at = $1, email = $2 WHERE id = $3", query)
	assert.Equal(t, []interface{}{time.Date(2021, time.December, 22, 20, 58, 0, 0, time.UTC), "test@example.com", 3}, args)
}

func TestDynamicUpdate3(t *testing.T) {
	var b endo.Builder
	patch := struct {
		ID       int
		Username string
		Email    string
		IP       string
	}{
		ID: 534,
		IP: "127.0.0.1",
	}
	patchField := func(cond bool, name string, v interface{}) {
		if cond {
			b.Write(", ").Write(name).WriteWithPlaced(" = ?", v)
		}
	}

	b.WriteWithArgs("UPDATE users SET updated_at = $1", time.Date(2021, 12, 22, 20, 58, 0, 0, time.UTC))

	patchField(patch.Username != "", "username", patch.Username)
	patchField(patch.Email != "", "email", patch.Email)
	patchField(patch.IP != "", "ip", patch.IP)

	b.WriteWithPlaced(" WHERE id = ?", 3)
	query, args := b.Build()

	assert.Equal(t, "UPDATE users SET updated_at = $1, ip = $2 WHERE id = $3", query)
	assert.Equal(t, []interface{}{time.Date(2021, time.December, 22, 20, 58, 0, 0, time.UTC), "127.0.0.1", 3}, args)
}

func TestCopy(t *testing.T) {
	var b endo.Builder
	b.WriteWithArgs("SELECT $1 AS marked, * FROM users", true)

	query1, args1 := b.Copy().
		WriteWithPlaced(" WHERE id = ?", 540).
		Build()
	query2, args2 := b.Copy().
		WriteWithPlaced(" WHERE username = ?", "admin").
		Build()

	assert.Equal(t, "SELECT $1 AS marked, * FROM users WHERE id = $2", query1)
	assert.Equal(t, []interface{}{true, 540}, args1)
	assert.Equal(t, "SELECT $1 AS marked, * FROM users WHERE username = $2", query2)
	assert.Equal(t, []interface{}{true, "admin"}, args2)
}
