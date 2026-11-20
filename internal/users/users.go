package users

import (
	"database/sql"
	"errors"

	"notaryos/internal/auth"
	"notaryos/internal/validate"
)

// User is an office account (staff) or portal client account.
type User struct {
	ID       int64
	OfficeID *int64
	Role     string
	Name     string
	Email    string
	Phone    string
	IsActive bool
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Authenticate verifies email/password for active users.
func (s *Store) Authenticate(email, password string) (User, error) {
	var u User
	var hash string
	var office sql.NullInt64
	var phone sql.NullString
	err := s.db.QueryRow(`SELECT u.id, u.office_id, r.name, u.name, u.email, u.phone, u.password_hash, u.is_active
		FROM users u JOIN roles r ON r.id=u.role_id WHERE u.email=?`, email).
		Scan(&u.ID, &office, &u.Role, &u.Name, &u.Email, &phone, &hash, &u.IsActive)
	if err == sql.ErrNoRows {
		return User{}, errors.New("email atau kata sandi salah")
	}
	if err != nil {
		return User{}, err
	}
	if !u.IsActive {
		return User{}, errors.New("akun dinonaktifkan")
	}
	if !auth.CheckPassword(hash, password) {
		return User{}, errors.New("email atau kata sandi salah")
	}
	if office.Valid {
		v := office.Int64
		u.OfficeID = &v
	}
	if phone.Valid {
		u.Phone = phone.String
	}
	_, _ = s.db.Exec(`UPDATE users SET last_login_at=UTC_TIMESTAMP() WHERE id=?`, u.ID)
	return u, nil
}

// Create registers a staff/client account with bcrypt-hashed password.
func (s *Store) Create(officeID *int64, role, name, email, phone, password string) (int64, error) {
	if err := validate.Required("nama", name); err != nil {
		return 0, err
	}
	if err := validate.Email(email); err != nil {
		return 0, err
	}
	h, err := auth.HashPassword(password)
	if err != nil {
		return 0, err
	}
	var roleID int64
	if err := s.db.QueryRow(`SELECT id FROM roles WHERE name=?`, role).Scan(&roleID); err != nil {
		return 0, errors.New("role tidak dikenal: " + role)
	}
	res, err := s.db.Exec(`INSERT INTO users(office_id, role_id, name, email, phone, password_hash)
		VALUES(?,?,?,?,?,?)`, officeID, roleID, name, email, phone, h)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// List returns office users ordered by name.
func (s *Store) List(officeID int64) ([]User, error) {
	rows, err := s.db.Query(`SELECT u.id, u.office_id, r.name, u.name, u.email, u.phone, u.is_active
		FROM users u JOIN roles r ON r.id=u.role_id WHERE u.office_id=? OR ? = 0 ORDER BY u.name`, officeID, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		var office sql.NullInt64
		var phone sql.NullString
		if err := rows.Scan(&u.ID, &office, &u.Role, &u.Name, &u.Email, &phone, &u.IsActive); err != nil {
			return nil, err
		}
		if office.Valid {
			v := office.Int64
			u.OfficeID = &v
		}
		if phone.Valid {
			u.Phone = phone.String
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
