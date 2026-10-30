package src

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

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
