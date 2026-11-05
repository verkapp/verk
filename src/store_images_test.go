package src

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

	withImage, err := st.importAttachment(task.ID, source)
	if err != nil {
		t.Fatalf("importAttachment: %v", err)
	}
	if len(withImage.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(withImage.Images))
	}
	image := withImage.Images[0]
	if image.Name != "pixel.png" {
		t.Fatalf("unexpected image name %q", image.Name)
	}
	if image.Mime != "image/png" {
		t.Fatalf("unexpected image mime %q", image.Mime)
	}
	if image.Size != int64(len(pngData)) {
		t.Fatalf("unexpected image size %d", image.Size)
	}
	if len(image.DataURL) < len("data:image/png;base64,") || image.DataURL[:len("data:image/png;base64,")] != "data:image/png;base64," {
		t.Fatalf("unexpected data URL prefix: %q", image.DataURL)
	}

	if _, err := st.deleteAttachment(image.ID); err != nil {
		t.Fatalf("deleteAttachment: %v", err)
	}
	reloaded, _ := st.getTask(task.ID)
	if len(reloaded.Images) != 0 {
		t.Fatal("image should be deleted")
	}

	if _, err := st.importAttachment(task.ID, source); err != nil {
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

func TestImportsPastedDocument(t *testing.T) {
	st := newTestStore(t)
	task, _ := st.createTask(TaskInput{Title: "Avec document collé"})
	pdf := []byte("%PDF-1.4\n% pièce jointe de test\n")

	withDocument, err := st.importAttachmentBytes(task.ID, "document", pdf)
	if err != nil {
		t.Fatalf("importAttachmentBytes: %v", err)
	}
	if len(withDocument.Images) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(withDocument.Images))
	}
	document := withDocument.Images[0]
	if document.Name != "document.pdf" {
		t.Fatalf("missing extension should be inferred, got %q", document.Name)
	}
	if document.Mime != "application/pdf" {
		t.Fatalf("unexpected attachment mime %q", document.Mime)
	}
	if document.DataURL != "" {
		t.Fatal("documents should not be embedded as data URLs")
	}
}

func TestAddsPastedAttachments(t *testing.T) {
	st := newTestStore(t)
	task, _ := st.createTask(TaskInput{Title: "Collage"})
	app := testApp(st)

	data := base64.StdEncoding.EncodeToString([]byte("bonjour"))
	updated, err := app.AddAttachments(task.ID, []AttachmentInput{
		{Name: "note.txt", Data: "data:text/plain;base64," + data},
	})
	if err != nil {
		t.Fatalf("AddAttachments: %v", err)
	}
	if len(updated.Images) != 1 {
		t.Fatalf("expected 1 pasted attachment, got %d", len(updated.Images))
	}
	attachment := updated.Images[0]
	if attachment.Name != "note.txt" || !strings.HasPrefix(attachment.Mime, "text/plain") {
		t.Fatalf("unexpected pasted attachment: %+v", attachment)
	}
}

func TestAcceptsDocumentAttachments(t *testing.T) {
	st := newTestStore(t)
	task, _ := st.createTask(TaskInput{Title: "Test"})
	source := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	withDocument, err := st.importAttachment(task.ID, source)
	if err != nil {
		t.Fatalf("importAttachment: %v", err)
	}
	if len(withDocument.Images) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(withDocument.Images))
	}
	document := withDocument.Images[0]
	if document.Name != "notes.txt" {
		t.Fatalf("unexpected attachment name %q", document.Name)
	}
	if !strings.HasPrefix(document.Mime, "text/plain") {
		t.Fatalf("unexpected attachment mime %q", document.Mime)
	}
	if document.Size != int64(len("hello")) {
		t.Fatalf("unexpected attachment size %d", document.Size)
	}
	if document.DataURL != "" {
		t.Fatal("documents should not be embedded as data URLs")
	}
}
