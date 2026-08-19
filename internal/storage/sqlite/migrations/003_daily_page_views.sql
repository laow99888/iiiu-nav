CREATE TABLE daily_page_views (
    day TEXT PRIMARY KEY CHECK (day GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    views INTEGER NOT NULL CHECK (views >= 0)
) STRICT, WITHOUT ROWID;
