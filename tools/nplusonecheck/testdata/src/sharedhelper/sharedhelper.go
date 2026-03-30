package sharedhelper

import "database/sql"

func loadUser(db *sql.DB, id int) {
	_ = db.QueryRow("SELECT id FROM users WHERE id = ?", id)
}

func Helper(db *sql.DB, id int) {
	loadUser(db, id)
}

func Noop(_ *sql.DB, _ int) {}
