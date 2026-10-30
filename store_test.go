package main

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *store {
	t.Helper()
	st, err := openStoreAt(t.TempDir())
	if err != nil {
		t.Fatalf("openStoreAt: %v", err)
	}
	t.Cleanup(func() {
		_ = st.close()
	})
	return st
}

func TestTaskLifecycle(t *testing.T) {
	st := newTestStore(t)
	due := time.Now().Add(2 * time.Hour).UnixMilli()

	task, err := st.createTask(TaskInput{
		Title:       "  Préparer   la démo ",
		Description: "  Relire le scénario  ",
		DueAt:       &due,
	})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	if task.Title != "Préparer la démo" {
		t.Fatalf("title not cleaned: %q", task.Title)
	}
	if task.Description != "Relire le scénario" {
		t.Fatalf("description not trimmed: %q", task.Description)
	}
	if task.Done || task.CompletedAt != nil {
		t.Fatal("new task should be open")
	}
	if task.DueAt == nil || *task.DueAt != due {
		t.Fatal("due date not persisted")
	}
	if len(task.Subtasks) != 0 || len(task.Images) != 0 {
		t.Fatal("new task should have no children")
	}

	updated, err := st.updateTask(task.ID, TaskInput{Title: "Démo finale", Description: "Trois slides", DueAt: nil})
	if err != nil {
		t.Fatalf("updateTask: %v", err)
	}
	if updated.Title != "Démo finale" || updated.DueAt != nil {
		t.Fatalf("update not applied: %+v", updated)
	}

	for _, title := range []string{"Écrire le fil rouge", "Répéter à voix haute"} {
		if _, err := st.addSubtask(task.ID, title); err != nil {
			t.Fatalf("addSubtask: %v", err)
		}
	}
	loaded, err := st.getTask(task.ID)
	if err != nil {
		t.Fatalf("getTask: %v", err)
	}
	if len(loaded.Subtasks) != 2 {
		t.Fatalf("expected 2 subtasks, got %d", len(loaded.Subtasks))
	}
	if loaded.Subtasks[0].Position != 0 || loaded.Subtasks[1].Position != 1 {
		t.Fatal("subtask positions not assigned")
	}

	first := loaded.Subtasks[0]
	if _, err := st.setSubtaskDone(first.ID, true); err != nil {
		t.Fatalf("setSubtaskDone: %v", err)
	}
	renamed, err := st.updateSubtask(first.ID, "Écrire le fil conducteur")
	if err != nil {
		t.Fatalf("updateSubtask: %v", err)
	}
	if renamed.Subtasks[0].Title != "Écrire le fil conducteur" || !renamed.Subtasks[0].Done {
		t.Fatal("subtask update not applied")
	}
	if _, err := st.deleteSubtask(first.ID); err != nil {
		t.Fatalf("deleteSubtask: %v", err)
	}

	done, err := st.setDone(task.ID, true)
	if err != nil {
		t.Fatalf("setDone: %v", err)
	}
	if !done.Done || done.CompletedAt == nil {
		t.Fatal("task not marked done with a completion date")
	}
	reopened, err := st.setDone(task.ID, false)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.Done || reopened.CompletedAt != nil {
		t.Fatal("reopening should clear completion date")
	}

	empty, err := st.createTask(TaskInput{Title: "   "})
	if err == nil || empty.ID != "" {
		t.Fatal("empty title should be rejected")
	}
}

func TestTaskOrdering(t *testing.T) {
	st := newTestStore(t)
	now := time.Now()
	past := now.Add(-time.Hour).UnixMilli()
	future := now.Add(48 * time.Hour).UnixMilli()

	undated, _ := st.createTask(TaskInput{Title: "Sans date"})
	late, _ := st.createTask(TaskInput{Title: "Plus tard", DueAt: &future})
	urgent, _ := st.createTask(TaskInput{Title: "Bientôt", DueAt: &past})
	finished, _ := st.createTask(TaskInput{Title: "Fini"})
	_, _ = st.setDone(finished.ID, true)

	tasks, err := st.listTasks()
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	if len(tasks) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != urgent.ID || tasks[1].ID != late.ID || tasks[2].ID != undated.ID {
		t.Fatalf("unexpected open task order: %s, %s, %s", tasks[0].Title, tasks[1].Title, tasks[2].Title)
	}
	if !tasks[3].Done || tasks[3].ID != finished.ID {
		t.Fatal("done tasks should be sorted last")
	}
}

func TestSettings(t *testing.T) {
	st := newTestStore(t)
	defaults, err := st.getSettings()
	if err != nil {
		t.Fatalf("getSettings: %v", err)
	}
	if defaults.Accent != "lavande" || defaults.CoachOnStart || defaults.ClaudePath != "" {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}

	want := Settings{Accent: "menthe", CoachOnStart: true, ClaudePath: "/opt/claude"}
	if err := st.saveSettings(want); err != nil {
		t.Fatalf("saveSettings: %v", err)
	}
	got, err := st.getSettings()
	if err != nil {
		t.Fatalf("getSettings: %v", err)
	}
	if got != want {
		t.Fatalf("settings mismatch: got %+v want %+v", got, want)
	}

	if err := st.saveSettings(Settings{}); err != nil {
		t.Fatalf("save empty settings: %v", err)
	}
	got, _ = st.getSettings()
	if got.Accent != "lavande" {
		t.Fatalf("empty accent should fall back to lavande, got %q", got.Accent)
	}
}

func TestImages(t *testing.T) {
	st := newTestStore(t)
	task, err := st.createTask(TaskInput{Title: "Avec image"})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}

	pngData, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "pixel.png")
	if err := os.WriteFile(source, pngData, 0o644); err != nil {
		t.Fatal(err)
	}

	withImage, err := st.importImage(task.ID, source)
	if err != nil {
		t.Fatalf("importImage: %v", err)
	}
	if len(withImage.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(withImage.Images))
	}
	image := withImage.Images[0]
	if image.Name != "pixel.png" {
		t.Fatalf("unexpected image name %q", image.Name)
	}
	if len(image.DataURL) < len("data:image/png;base64,") || image.DataURL[:len("data:image/png;base64,")] != "data:image/png;base64," {
		t.Fatalf("unexpected data URL prefix: %q", image.DataURL)
	}

	if _, err := st.deleteImage(image.ID); err != nil {
		t.Fatalf("deleteImage: %v", err)
	}
	reloaded, _ := st.getTask(task.ID)
	if len(reloaded.Images) != 0 {
		t.Fatal("image should be deleted")
	}

	if _, err := st.importImage(task.ID, source); err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if err := st.deleteTask(task.ID); err != nil {
		t.Fatalf("deleteTask: %v", err)
	}
	if _, err := st.getTask(task.ID); !errors.Is(err, errNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	entries, err := os.ReadDir(st.imageDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("image files should be removed with the task, %d left", len(entries))
	}
}

func TestRejectsNonImage(t *testing.T) {
	st := newTestStore(t)
	task, _ := st.createTask(TaskInput{Title: "Test"})
	source := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := st.importImage(task.ID, source); err == nil {
		t.Fatal("non-image files should be rejected")
	}
}
