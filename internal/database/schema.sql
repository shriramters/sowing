-- SOWING Database Schema

-- Silos are the top-level content areas.
CREATE TABLE silos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    archived_at TIMESTAMP
);

-- Users are the authors of content.
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL
);

-- Identities provide a way for users to authenticate.
CREATE TABLE identities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    provider TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    password_hash TEXT,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- Pages are the individual wiki pages.
CREATE TABLE pages (
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
CREATE TABLE revisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    author_id INTEGER NOT NULL,
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(page_id) REFERENCES pages(id),
    FOREIGN KEY(author_id) REFERENCES users(id)
);
