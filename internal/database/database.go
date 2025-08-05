package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func New(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
-- SOWING Database Schema

-- Silos are the top-level content areas.
CREATE TABLE IF NOT EXISTS silos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    archived_at TIMESTAMP,
    cover_image TEXT
);

-- Users are the authors of content.
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL
);

-- Identities provide a way for users to authenticate.
CREATE TABLE IF NOT EXISTS identities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    provider TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    password_hash TEXT,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- Pages are the individual wiki pages.
CREATE TABLE IF NOT EXISTS pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    silo_id INTEGER NOT NULL,
    parent_id INTEGER,
    slug TEXT NOT NULL,
    title TEXT NOT NULL,
    current_revision_id INTEGER NOT NULL,
    archived_at TIMESTAMP,
    FOREIGN KEY(silo_id) REFERENCES silos(id),
    FOREIGN KEY(parent_id) REFERENCES pages(id),
    UNIQUE (silo_id, parent_id, slug)
);

-- Revisions are the history of a page.
CREATE TABLE IF NOT EXISTS revisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    author_id INTEGER NOT NULL,
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(page_id) REFERENCES pages(id),
    FOREIGN KEY(author_id) REFERENCES users(id)
);
`)
	return err
}
