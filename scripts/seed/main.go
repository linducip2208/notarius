// Command seed creates the office + first super-admin for development.
// All values come from env; never hardcode credentials.
//
//	SEED_OFFICE, SEED_NOTARY, SEED_SK, SEED_ADDR, SEED_PHONE, SEED_EMAIL,
//	SEED_ADMIN_NAME, SEED_ADMIN_EMAIL, SEED_ADMIN_PASS (+ standard DB_* env)
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"notaryos/internal/auth"
	"notaryos/internal/config"
)

func main() {
	cfg := config.Load()
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	office := env("SEED_OFFICE", "Kantor Notaris Contoh")
	res, err := db.Exec(`INSERT INTO offices(name, notary_name, sk_number, address, phone, email)
		VALUES(?,?,?,?,?,?)`, office, env("SEED_NOTARY", "Nama Notaris, S.H."),
		env("SEED_SK", "SK-000"), env("SEED_ADDR", "Jl. Contoh 1"),
		env("SEED_PHONE", "021-000"), env("SEED_EMAIL", "office@contoh.id"))
	if err != nil {
		log.Fatal(err)
	}
	oid, _ := res.LastInsertId()
	var roleID int64
	if err := db.QueryRow(`SELECT id FROM roles WHERE name='super_admin'`).Scan(&roleID); err != nil {
		log.Fatal(err)
	}
	h, err := auth.HashPassword(env("SEED_ADMIN_PASS", "Admin123!"))
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users(office_id, role_id, name, email, password_hash)
		VALUES(?,?,?,?,?)`, oid, roleID, env("SEED_ADMIN_NAME", "Administrator"),
		env("SEED_ADMIN_EMAIL", "admin@contoh.id"), h)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("seeded office", oid, "admin", env("SEED_ADMIN_EMAIL", "admin@contoh.id"))
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
