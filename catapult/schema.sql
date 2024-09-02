-- catapult/schema.sql
CREATE TABLE IF NOT EXISTS file_sizes (
    file_path TEXT PRIMARY KEY,
    size INTEGER
);

CREATE TABLE IF NOT EXISTS copied_files (
    file_path TEXT,
    destination TEXT,
    PRIMARY KEY (file_path, destination)
);