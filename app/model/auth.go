package model

const (
	RoleAdmin     = "admin"
	RolePengelola = "pengelola"
	RolePeminjam  = "peminjam"
)

// AuthUser adalah identitas user yang diambil dari JWT.
type AuthUser struct {
	ID    int
	Email string
	Role  string
}
