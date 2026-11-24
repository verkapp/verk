package src

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyDataDirMovesTodoFolder(t *testing.T) {
	base := t.TempDir()
	legacy := filepath.Join(base, legacyDataDirName)
	if err := os.MkdirAll(filepath.Join(legacy, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, legacyDatabaseName), []byte("db"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(base, dataDirName)
	if err := migrateLegacyDataDir(legacy, dir); err != nil {
		t.Fatalf("migrateLegacyDataDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, legacyDatabaseName)); err != nil {
		t.Fatalf("database should have moved: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy folder should be gone, got %v", err)
	}
}

func TestMigrateLegacyDataDirLeavesUnrelatedFolders(t *testing.T) {
	base := t.TempDir()
	legacy := filepath.Join(base, legacyDataDirName)
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(base, dataDirName)
	if err := migrateLegacyDataDir(legacy, dir); err != nil {
		t.Fatalf("migrateLegacyDataDir: %v", err)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatalf("a folder without todo.db must stay in place: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("nothing should be created, got %v", err)
	}
}

func TestMigrateLegacyDataDirKeepsExistingVerkFolder(t *testing.T) {
	base := t.TempDir()
	legacy := filepath.Join(base, legacyDataDirName)
	dir := filepath.Join(base, dataDirName)
	for _, d := range []string{legacy, dir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(legacy, legacyDatabaseName), []byte("db"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := migrateLegacyDataDir(legacy, dir); err != nil {
		t.Fatalf("migrateLegacyDataDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacy, legacyDatabaseName)); err != nil {
		t.Fatalf("legacy data must not be touched when verk already exists: %v", err)
	}
}

func TestOpenStoreRenamesLegacyDatabase(t *testing.T) {
	dir := t.TempDir()
	first, err := openStoreAt(dir)
	if err != nil {
		t.Fatalf("openStoreAt: %v", err)
	}
	if _, err := first.createProfile("Perso"); err != nil {
		t.Fatalf("createProfile: %v", err)
	}
	if err := first.close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, databaseName), filepath.Join(dir, legacyDatabaseName)); err != nil {
		t.Fatal(err)
	}

	st, err := openStoreAt(dir)
	if err != nil {
		t.Fatalf("openStoreAt after rename: %v", err)
	}
	t.Cleanup(func() { _ = st.close() })
	if _, err := os.Stat(filepath.Join(dir, legacyDatabaseName)); !os.IsNotExist(err) {
		t.Fatalf("todo.db should have been renamed, got %v", err)
	}
	profiles, err := st.listProfiles()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range profiles {
		if p.Name == "Perso" {
			found = true
		}
	}
	if !found {
		t.Fatalf("data from todo.db should be kept, got %+v", profiles)
	}
}
