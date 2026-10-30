package src

const schema = `
	PRAGMA journal_mode = WAL;
	PRAGMA foreign_keys = ON;
	PRAGMA busy_timeout = 5000;
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		done INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		due_at INTEGER,
		completed_at INTEGER
	);
	CREATE TABLE IF NOT EXISTS subtasks (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		done INTEGER NOT NULL DEFAULT 0,
		position INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS images (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		file TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_subtasks_task ON subtasks(task_id);
	CREATE INDEX IF NOT EXISTS idx_images_task ON images(task_id);
`
