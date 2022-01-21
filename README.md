# Endo

🚧 WORK IN PROGRESS 🚧

Endo generates CRUD functions (SQL) for your Go structs, in a bring-your-own-types manner. Here's how it works:
- You write your models in Go structs using your types.
- You run `endogen` to generate the CRUD functions (Get, GetAll, Create, Update, Patch, Delete) for those models.
- You can use and extend the generated code.

An example model:

```go
import "gopkg.in/guregu/null.v4"

type User struct {
	ID                int         `db:"id,primary"`
	Email             string      `db:"email"`
	FirstName         null.String `db:"first_name"`
	LastName          null.String `db:"last_name"`
	EmailVerified     bool        `db:"email_verified"`
	PasswordHash      null.String `db:"password_hash"`
	CreatedAt         time.Time   `db:"created_at"`
	UpdatedAt         time.Time   `db:"updated_at"`
}
```

Endo generates the following functions for the model:

```go
GetUser(ctx context.Context, key int) (*User, error)
GetUserByField(ctx context.Context, field string, v interface{}) (*User, error)
GetUsers(ctx context.Context, po endo.PageOptions) ([]*User, error)
CreateUser(ctx context.Context, e User) (*User, error)
UpdateUser(ctx context.Context, key int, e User) (*User, error)
UpdateUserByField(ctx context.Context, field string, v interface{}, e User) (*User, error)
PatchUser(ctx context.Context, key int, p UserPatch) (*User, error)
PatchUserByField(ctx context.Context, field string, v interface{}, p UserPatch) (*User, error)
DeleteUser(ctx context.Context, key int) error
DeleteUserByField(ctx context.Context, field string, v interface{}) error
```

Checkout the `examples` directory for more.

## Features

- Basic CRUD functions (with SQL) based on Go structs. Supports all your types!
- Additional functions when a `primary` key is used.
- Patches using dynamic SQL.
  > _Endo has an simple builtin query generator `endo.Builder`, it's actually `strings.Builder` with a few additions._
- Supports transactional contexts through `endo.TxFunc`.
- Optional customization via comment parameters.
- Extensible and reusable.

## Why another library like x?

Although [sqlx](https://github.com/jmoiron/sqlx), [sqlc](https://github.com/kyleconroy/sqlc) and [sqlboiler](https://github.com/volatiletech/sqlboiler) are great libraries with wide support by multiple communities, it always feels like some important features are missing like simple dynamic patching. Endo doesn't try to replace those libraries, it simply tries a different approach to cover the basic things. If you don't need dynamic patching or extensibility, you're better off with sqlc or sqlx. Endo tries to be as simple and boring as possible, yet extensible and easy-to-use, less is more.

## Roadmap

- [x] Basic CRUD templates.
- [x] Supports PostgreSQL.
- [x] Dynamic patches using dynamic SQL generation with minimal overhead.
- [ ] MySQL support.
