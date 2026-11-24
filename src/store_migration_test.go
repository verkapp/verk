package src

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// legacySchema is the tasks schema shipped with v1.1.0. Upgrading from it used
// to fail because the new schema created an index on tasks(profile_id) before
// the migration had a chance to add the column.
const legacySchema = `
	CREATE TABLE tasks (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		done INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		due_at INTEGER,
		completed_at INTEGER,
		started_at INTEGER,
		time_spent INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE subtasks (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		done INTEGER NOT NULL DEFAULT 0,
		position INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE images (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		file TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE tombstones (
		kind TEXT NOT NULL,
		id TEXT NOT NULL,
		task_id TEXT NOT NULL DEFAULT '',
		deleted_at INTEGER NOT NULL,
		PRIMARY KEY (kind, id)
	);
	CREATE TABLE settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
`

func TestOpenStoreMigratesLegacyDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, databaseName)

	old, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	if _, err := old.Exec(legacySchema); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	if _, err := old.Exec(`INSERT INTO tasks (id, title, created_at) VALUES ('t1', 'ancienne tâche', 1000)`); err != nil {
		t.Fatalf("insert legacy task: %v", err)
	}
	if err := old.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	st, err := openStoreAt(dir)
	if err != nil {
		t.Fatalf("openStoreAt should migrate the legacy database: %v", err)
	}
	t.Cleanup(func() { _ = st.close() })

	columns, err := tableColumns(st.db, "tasks")
	if err != nil {
		t.Fatalf("tableColumns: %v", err)
	}
	if !columns["profile_id"] {
		t.Fatal("profile_id should have been added by the migration")
	}

	tasks, err := st.listTasks()
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "t1" {
		t.Fatalf("legacy task should be preserved, got %+v", tasks)
	}
	if tasks[0].ProfileID == "" {
		t.Fatal("legacy task should be attached to the fallback profile")
	}
}
