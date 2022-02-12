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

func TestWritef(t *testing.T) {
	var b endo.Builder

	query := b.Writef("SELECT * FROM %s", "users").String()

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

func TestWriteWithQuestionMarkParams(t *testing.T) {
	b := endo.Builder{FormatParam: endo.QuestionMarkParam}

	query, args := b.WriteTrim(`
			SELECT id, username, email FROM users
		`).
		WriteWithParams(" WHERE id = {}", 10).
		Build()

	assert.Equal(t, "SELECT id, username, email FROM users WHERE id = ?", query)
	assert.Equal(t, []interface{}{10}, args)
}

func TestSQLSelect(t *testing.T) {
	var b endo.Builder

	query, args := b.WriteTrim(`
			SELECT id, username, email FROM users
		`).
		WriteWithParams(" WHERE id = {}", 10).
		Build()

	assert.Equal(t, "SELECT id, username, email FROM users WHERE id = $1", query)
	assert.Equal(t, []interface{}{10}, args)
}

func TestSQLSelectMultipleParams(t *testing.T) {
	var b endo.Builder

	query, args := b.WriteTrim(`
			SELECT id, username, email FROM users
		`).
		WriteWithParams(
			" WHERE active = {} ORDER BY id LIMIT {} OFFSET 0",
			true,
			100,
		).
		Build()

	assert.Equal(t, "SELECT id, username, email FROM users WHERE active = $1 ORDER BY id LIMIT $2 OFFSET 0", query)
	assert.Equal(t, []interface{}{true, 100}, args)
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
		b.WriteWithParams(" = {}", v)
	}
	b.WriteWithParams(" WHERE id = {}", 3)
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
		b.WriteWithParams(fmt.Sprintf(", %s = {}", k), v)
	}
	b.WriteWithParams(" WHERE id = {}", 3)
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
			b.Write(", ").Write(name).WriteWithParams(" = {}", v)
		}
	}

	b.WriteWithArgs("UPDATE users SET updated_at = $1", time.Date(2021, 12, 22, 20, 58, 0, 0, time.UTC))

	patchField(patch.Username != "", "username", patch.Username)
	patchField(patch.Email != "", "email", patch.Email)
	patchField(patch.IP != "", "ip", patch.IP)

	b.WriteWithParams(" WHERE id = {}", 3)
	query, args := b.Build()

	assert.Equal(t, "UPDATE users SET updated_at = $1, ip = $2 WHERE id = $3", query)
	assert.Equal(t, []interface{}{time.Date(2021, time.December, 22, 20, 58, 0, 0, time.UTC), "127.0.0.1", 3}, args)
}

func TestUpdateWithNamedArgs(t *testing.T) {
	var b endo.Builder
	patch := []endo.NamedArg{
		{"email", "test@example.com"},
		{"ip", "127.0.0.1"},
		{"log_id", 38154},
	}

	query, args := b.
		Write("UPDATE users SET ").
		WriteNamedArgs("%s = {}", ", ", patch...).
		WriteWithParams(" WHERE id = {}", 3).
		Build()

	assert.Equal(t, "UPDATE users SET email = $1, ip = $2, log_id = $3 WHERE id = $4", query)
	assert.Equal(t, []interface{}{"test@example.com", "127.0.0.1", 38154, 3}, args)
}

func TestDynamicFilters(t *testing.T) {
	var b endo.Builder
	filters := []endo.NamedArg{
		{"email = {}", "test@example.com"},
		{"(ip = {} OR is_external)", "127.0.0.1"},
		{Name: "active"},
	}

	query, args := b.
		Write("SELECT * FROM users WHERE ").
		WriteNamedArgs("%s", " AND ", filters...).
		Build()

	assert.Equal(t, "SELECT * FROM users WHERE email = $1 AND (ip = $2 OR is_external) AND active", query)
	assert.Equal(t, []interface{}{"test@example.com", "127.0.0.1"}, args)
}

func TestDynamicFiltersArgs(t *testing.T) {
	var b endo.Builder
	filters := []endo.NamedArg{
		{"email = {}", "test@example.com"},
		{"manager_id = {} OR manager_id = {}", endo.Args{765, 92}},
		{"ip = {} OR is_external", "127.0.0.1"},
		{Name: "active"},
	}

	query, args := b.
		Write("SELECT * FROM users WHERE ").
		WriteNamedArgs("(%s)", " AND ", filters...).
		Build()

	assert.Equal(t, "SELECT * FROM users WHERE (email = $1) AND (manager_id = $2 OR manager_id = $3) AND (ip = $4 OR is_external) AND (active)", query)
	assert.Equal(t, []interface{}{"test@example.com", 765, 92, "127.0.0.1"}, args)
}

func TestCopy(t *testing.T) {
	var b endo.Builder
	b.WriteWithArgs("SELECT $1 AS marked, * FROM users", true)

	query1, args1 := b.Copy().
		WriteWithParams(" WHERE id = {}", 540).
		Build()
	query2, args2 := b.Copy().
		WriteWithParams(" WHERE username = {}", "admin").
		Build()

	assert.Equal(t, "SELECT $1 AS marked, * FROM users WHERE id = $2", query1)
	assert.Equal(t, []interface{}{true, 540}, args1)
	assert.Equal(t, "SELECT $1 AS marked, * FROM users WHERE username = $2", query2)
	assert.Equal(t, []interface{}{true, "admin"}, args2)
}
