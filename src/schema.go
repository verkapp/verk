package src

const schema = `
	PRAGMA journal_mode = WAL;
	PRAGMA foreign_keys = ON;
	PRAGMA busy_timeout = 5000;
	CREATE TABLE IF NOT EXISTS profiles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		color TEXT NOT NULL DEFAULT '#7b6cf0',
		position INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL DEFAULT 0,
		auto_defer_at TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		profile_id TEXT NOT NULL DEFAULT '',
		title TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		done INTEGER NOT NULL DEFAULT 0,
		wont_do INTEGER NOT NULL DEFAULT 0,
		position INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		due_at INTEGER,
		completed_at INTEGER,
		started_at INTEGER,
		time_spent INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS subtasks (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		done INTEGER NOT NULL DEFAULT 0,
		position INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS images (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		file TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS tombstones (
		kind TEXT NOT NULL,
		id TEXT NOT NULL,
		task_id TEXT NOT NULL DEFAULT '',
		deleted_at INTEGER NOT NULL,
		PRIMARY KEY (kind, id)
	);
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
`

// indexes are created after migrate: an index may reference a column that only
// exists in recent databases (profile_id), and creating it before the ALTER
// TABLE that adds the column made older databases fail to open.
const indexes = `
	CREATE INDEX IF NOT EXISTS idx_subtasks_task ON subtasks(task_id);
	CREATE INDEX IF NOT EXISTS idx_images_task ON images(task_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_profile ON tasks(profile_id);
`
