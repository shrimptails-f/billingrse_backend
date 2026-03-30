package sharedmethod

import "database/sql"

type Repo struct {
	DB *sql.DB
}

func (r *Repo) loadUser(id int) {
	r.queryUser(id)
}

func (r *Repo) queryUser(id int) {
	_ = r.DB.QueryRow("SELECT id FROM users WHERE id = ?", id)
}

func (r *Repo) LoadUser(id int) {
	r.loadUser(id)
}

func (r *Repo) Touch(_ int) {}
