package src

import (
	"testing"
	"time"
)

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

func TestCompletingTaskCompletesSubtasks(t *testing.T) {
	st := newTestStore(t)
	task, err := st.createTask(TaskInput{Title: "Préparer le lancement"})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	for _, title := range []string{"Écrire l’annonce", "Relire la page", "Envoyer l’e-mail"} {
		if _, err := st.addSubtask(task.ID, title); err != nil {
			t.Fatalf("addSubtask: %v", err)
		}
	}

	done, err := st.setDone(task.ID, true)
	if err != nil {
		t.Fatalf("setDone: %v", err)
	}
	if !done.Done {
		t.Fatal("task should be done")
	}
	for _, subtask := range done.Subtasks {
		if !subtask.Done {
			t.Fatalf("subtask %q should be done with its task", subtask.Title)
		}
	}

	reopened, err := st.setDone(task.ID, false)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.Done {
		t.Fatal("task should be reopened")
	}
}

func TestTimeTracking(t *testing.T) {
	st := newTestStore(t)
	task, err := st.createTask(TaskInput{Title: "chrono"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.setStarted(task.ID, true); err != nil {
		t.Fatal(err)
	}
	// Backdate the timer by one minute, then pause.
	if _, err := st.db.Exec(`UPDATE tasks SET started_at = started_at - 60000 WHERE id = ?`, task.ID); err != nil {
		t.Fatal(err)
	}
	got, err := st.setStarted(task.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.StartedAt != nil || got.TimeSpent < 60000 {
		t.Fatalf("pause should bank the time: %+v", got)
	}

	// Finishing a running task banks its time too.
	if _, err := st.setStarted(task.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`UPDATE tasks SET started_at = started_at - 60000 WHERE id = ?`, task.ID); err != nil {
		t.Fatal(err)
	}
	done, err := st.setDone(task.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if done.StartedAt != nil || done.TimeSpent < 120000 {
		t.Fatalf("done should bank the time: %+v", done)
	}
}
