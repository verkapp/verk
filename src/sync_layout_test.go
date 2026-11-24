package src

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// layoutDataset has two profiles, tasks with subtasks and images, and a
// tombstone: every kind of entity the layout spreads over files.
func layoutDataset() snapshot {
	s := emptySnapshot()
	s.Profiles = []snapshotProfile{
		{ID: "p-perso", Name: "Perso", Color: "#7b6cf0", Position: 0, CreatedAt: 1, UpdatedAt: 1},
		{ID: "p-travail", Name: "Travail", Color: "#4aa985", Position: 1, CreatedAt: 2, UpdatedAt: 2, AutoDeferAt: "18:00"},
	}
	due := int64(5000)
	s.Tasks = []snapshotTask{
		{ID: "t-1", ProfileID: "p-perso", Title: "Acheter du pain", CreatedAt: 10, UpdatedAt: 11, DueAt: &due},
		{ID: "t-2", ProfileID: "p-travail", Title: "Préparer la démo", Description: "Dix minutes", CreatedAt: 12, UpdatedAt: 13, TimeSpent: 60},
		{ID: "t-3", ProfileID: "p-travail", Title: "Envoyer le compte rendu", Done: true, CreatedAt: 14, UpdatedAt: 15},
	}
	s.Subtasks = []snapshotSubtask{
		{ID: "s-1", TaskID: "t-2", Title: "Relire", Position: 0, UpdatedAt: 20},
		{ID: "s-2", TaskID: "t-2", Title: "Répéter", Done: true, Position: 1, UpdatedAt: 21},
	}
	s.Images = []snapshotImage{
		{ID: "i-1", TaskID: "t-2", Name: "schema.png", File: "i-1.png", CreatedAt: 30, UpdatedAt: 30},
	}
	s.Tombstones = []snapshotTombstone{{Kind: tombstoneTask, ID: "t-old", DeletedAt: time.Now().UnixMilli()}}
	return s
}

func requireSameDataset(t *testing.T, got, want snapshot) {
	t.Helper()
	if !snapshotsEqual(sortSnapshot(got), sortSnapshot(want)) {
		gotJSON, _ := json.MarshalIndent(got.body(), "", " ")
		wantJSON, _ := json.MarshalIndent(want.body(), "", " ")
		t.Fatalf("datasets differ\n got: %s\nwant: %s", gotJSON, wantJSON)
	}
}

func readJSONFile(t *testing.T, path string, into any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func TestSnapshotTreeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := layoutDataset()
	if err := writeSnapshotTree(dir, want); err != nil {
		t.Fatalf("writeSnapshotTree: %v", err)
	}

	for _, rel := range []string{
		"profiles/p-perso/profile.json",
		"profiles/p-perso/tasks/t-1.json",
		"profiles/p-travail/profile.json",
		"profiles/p-travail/tasks/t-2.json",
		"profiles/p-travail/tasks/t-3.json",
		"tombstones.json",
		"todo.json",
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}

	var task taskFile
	readJSONFile(t, filepath.Join(dir, "profiles", "p-travail", "tasks", "t-2.json"), &task)
	if task.Title != "Préparer la démo" || len(task.Subtasks) != 2 || len(task.Images) != 1 {
		t.Fatalf("t-2.json should hold the task, its subtasks and its image, got %+v", task)
	}

	got, err := readSnapshotTree(dir)
	if err != nil {
		t.Fatalf("readSnapshotTree: %v", err)
	}
	requireSameDataset(t, got, want)
}

func TestSnapshotTreeLeavesUnchangedFilesAlone(t *testing.T) {
	dir := t.TempDir()
	snap := layoutDataset()
	if err := writeSnapshotTree(dir, snap); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "profiles", "p-perso", "tasks", "t-1.json")
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}

	snap.Tasks[1].Title = "Préparer la démo, version 2"
	if err := writeSnapshotTree(dir, snap); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(old) {
		t.Fatalf("an unchanged task file must not be rewritten")
	}
}

func TestSnapshotTreeRemovesStaleFiles(t *testing.T) {
	dir := t.TempDir()
	snap := layoutDataset()
	if err := writeSnapshotTree(dir, snap); err != nil {
		t.Fatal(err)
	}

	// t-3 is deleted, t-1 moves to Travail, and Perso disappears.
	snap.Tasks = []snapshotTask{snap.Tasks[0], snap.Tasks[1]}
	snap.Tasks[0].ProfileID = "p-travail"
	snap.Profiles = snap.Profiles[1:]
	if err := writeSnapshotTree(dir, snap); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "profiles", "p-travail", "tasks", "t-3.json")); !os.IsNotExist(err) {
		t.Fatalf("a deleted task's file should be removed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "profiles", "p-travail", "tasks", "t-1.json")); err != nil {
		t.Fatalf("a moved task should be written in its new profile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "profiles", "p-perso")); !os.IsNotExist(err) {
		t.Fatalf("an empty profile folder should be removed, got %v", err)
	}
	got, err := readSnapshotTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	requireSameDataset(t, got, snap)
}

// legacyTodoJSON is todo.json as Verk 2.0 and Todo wrote it (format 2).
func legacyTodoJSON(t *testing.T, snap snapshot) []byte {
	t.Helper()
	snap.Version = 2
	data, err := json.MarshalIndent(snap.body(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}

func TestReadRepoSnapshotDetectsLegacyFormat(t *testing.T) {
	dir := t.TempDir()
	want := layoutDataset()
	if err := os.WriteFile(filepath.Join(dir, snapshotFileName), legacyTodoJSON(t, want), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readRepoSnapshot(dir, time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("readRepoSnapshot: %v", err)
	}
	if !got.legacy {
		t.Fatalf("a format 2 todo.json must be flagged for migration")
	}
	requireSameDataset(t, got.snapshot, want)
}

func TestReadRepoSnapshotMergesLegacyIntoTree(t *testing.T) {
	dir := t.TempDir()
	tree := layoutDataset()
	if err := writeSnapshotTree(dir, tree); err != nil {
		t.Fatal(err)
	}
	// An older device pushed one last todo.json with a task of its own.
	legacy := layoutDataset()
	legacy.Tasks = append(legacy.Tasks, snapshotTask{ID: "t-4", ProfileID: "p-perso", Title: "Depuis la 2.0", CreatedAt: 40, UpdatedAt: 40})
	if err := os.WriteFile(filepath.Join(dir, snapshotFileName), legacyTodoJSON(t, legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readRepoSnapshot(dir, time.Now().UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	if !got.legacy || len(got.Tasks) != 4 {
		t.Fatalf("the legacy task must be merged with the tree, got legacy=%v tasks=%d", got.legacy, len(got.Tasks))
	}
}

func TestReadRepoSnapshotRejectsNewerFormat(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, snapshotFileName), []byte(`{"version": 4}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readRepoSnapshot(dir, 0); err == nil || !strings.Contains(err.Error(), "plus récente") {
		t.Fatalf("a newer format must be refused, got %v", err)
	}
}

// The fence only works if the reader of Verk 2.0 refuses it. decodeSnapshot
// is that reader, unchanged.
func TestLayoutFenceStopsFormat2Readers(t *testing.T) {
	data, err := json.Marshal(newLayoutFence())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeSnapshot(data); err == nil || !strings.Contains(err.Error(), "plus récente") {
		t.Fatalf("Verk 2.0 must refuse the fence, got %v", err)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// TestSyncMigratesLegacyRepository plays the migration end to end: a remote
// filled by Verk 2.0, then the first sync of this version.
func TestSyncMigratesLegacyRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	runTestCommand(t, "git", "init", "--bare", "-b", "main", remote)

	// What Verk 2.0 left in the repository.
	legacy := layoutDataset()
	legacy.Images = nil // the attachment files are not part of this test
	seed := filepath.Join(t.TempDir(), "seed")
	runTestCommand(t, "git", "clone", remote, seed)
	if err := os.WriteFile(filepath.Join(seed, snapshotFileName), legacyTodoJSON(t, legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOutput(t, seed, "-c", "user.name=Verk 2.0", "-c", "user.email=v2@localhost", "add", "-A")
	gitOutput(t, seed, "-c", "user.name=Verk 2.0", "-c", "user.email=v2@localhost", "commit", "-m", "sync: verk 2.0")
	gitOutput(t, seed, "push", "origin", "HEAD:main")

	st := newTestStore(t)
	app := testApp(st)
	cfg := syncConfig{Remote: remote, Enabled: true, Branch: "main"}
	sync := func() {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := app.performSync(ctx, cfg); err != nil {
			t.Fatalf("performSync: %v", err)
		}
	}
	sync()

	tasks, err := st.listTasks()
	if err != nil {
		t.Fatal(err)
	}
	titles := map[string]bool{}
	for _, task := range tasks {
		titles[task.Title] = true
	}
	for _, task := range legacy.Tasks {
		if !titles[task.Title] {
			t.Fatalf("task %q from the legacy todo.json is missing locally: %v", task.Title, titles)
		}
	}

	// The remote now holds the folder layout and the fence.
	check := filepath.Join(t.TempDir(), "check")
	runTestCommand(t, "git", "clone", remote, check)
	var fence layoutFence
	readJSONFile(t, filepath.Join(check, snapshotFileName), &fence)
	if fence.Version != syncFormatVersion {
		t.Fatalf("todo.json should become the fence, got %+v", fence)
	}
	var task taskFile
	readJSONFile(t, filepath.Join(check, "profiles", "p-travail", "tasks", "t-2.json"), &task)
	if task.Title != "Préparer la démo" || len(task.Subtasks) != 2 {
		t.Fatalf("t-2 should be migrated with its subtasks, got %+v", task)
	}
	log := gitOutput(t, check, "log", "--format=%s")
	if !strings.Contains(log, migrationCommitMessage) {
		t.Fatalf("the migration commit should be named, got log:\n%s", log)
	}

	// A second sync has nothing to migrate and nothing to commit.
	before := gitOutput(t, check, "rev-parse", "HEAD")
	sync()
	gitOutput(t, check, "pull", "--ff-only")
	if after := gitOutput(t, check, "rev-parse", "HEAD"); after != before {
		t.Fatalf("a sync without changes should not commit, %s -> %s", before, after)
	}
}
