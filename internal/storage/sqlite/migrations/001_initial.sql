CREATE TABLE categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    slug TEXT NOT NULL UNIQUE CHECK (length(trim(slug)) > 0),
    icon_name TEXT NOT NULL DEFAULT '',
    visibility TEXT NOT NULL CHECK (visibility IN ('public', 'private')),
    sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;

CREATE INDEX categories_visibility_order_idx
    ON categories (visibility, sort_order, id);

CREATE TABLE links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    description TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL CHECK (length(trim(url)) > 0),
    icon_source TEXT NOT NULL CHECK (icon_source IN ('auto', 'url', 'upload', 'generated')),
    icon_value TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;

CREATE INDEX links_category_order_idx
    ON links (category_id, sort_order, id);

CREATE TABLE settings (
    key TEXT PRIMARY KEY CHECK (length(trim(key)) > 0),
    value_json TEXT NOT NULL CHECK (json_valid(value_json))
) STRICT, WITHOUT ROWID;

CREATE TABLE admin (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    password_hash TEXT NOT NULL CHECK (length(password_hash) > 0),
    updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE sessions (
    token_hash BLOB PRIMARY KEY CHECK (length(token_hash) = 32),
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL CHECK (expires_at > created_at)
) STRICT, WITHOUT ROWID;

CREATE INDEX sessions_expiry_idx ON sessions (expires_at);
