package src

import (
	"testing"
	"time"
)

func TestProfileLifecycle(t *testing.T) {
	st := newTestStore(t)
	profiles, err := st.listProfiles()
	if err != nil {
		t.Fatalf("listProfiles: %v", err)
	}
	if len(profiles) < 2 {
		t.Fatalf("starter profiles missing: %+v", profiles)
	}

	profile, err := st.createProfile("  Créatif  ")
	if err != nil {
		t.Fatalf("createProfile: %v", err)
	}
	if profile.Name != "Créatif" || profile.Color == "" {
		t.Fatalf("unexpected profile: %+v", profile)
	}

	task, err := st.createTask(TaskInput{Title: "Croquis", ProfileID: profile.ID})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	if task.ProfileID != profile.ID {
		t.Fatalf("task should live in the new profile, got %q", task.ProfileID)
	}

	updated, err := st.updateProfile(profile.ID, "Idées", "#123456")
	if err != nil {
		t.Fatalf("updateProfile: %v", err)
	}
	if updated.Name != "Idées" || updated.Color != "#123456" {
		t.Fatalf("profile update not applied: %+v", updated)
	}

	if err := st.deleteProfile(profile.ID); err != nil {
		t.Fatalf("deleteProfile: %v", err)
	}
	moved, err := st.getTask(task.ID)
	if err != nil {
		t.Fatalf("getTask: %v", err)
	}
	if moved.ProfileID == profile.ID || moved.ProfileID == "" {
		t.Fatalf("tasks should move to a remaining profile, got %q", moved.ProfileID)
	}

	// The last profile is protected: there must always be a container.
	remaining, err := st.listProfiles()
	if err != nil {
		t.Fatalf("listProfiles: %v", err)
	}
	for len(remaining) > 1 {
		if err := st.deleteProfile(remaining[len(remaining)-1].ID); err != nil {
			t.Fatalf("deleteProfile: %v", err)
		}
		remaining, err = st.listProfiles()
		if err != nil {
			t.Fatalf("listProfiles: %v", err)
		}
	}
	if err := st.deleteProfile(remaining[0].ID); err == nil {
		t.Fatal("deleting the last profile should fail")
	}
}

func TestReorderTasks(t *testing.T) {
	st := newTestStore(t)
	profiles, err := st.listProfiles()
	if err != nil {
		t.Fatal(err)
	}
	profileID := profiles[0].ID

	first, _ := st.createTask(TaskInput{Title: "Un", ProfileID: profileID})
	second, _ := st.createTask(TaskInput{Title: "Deux", ProfileID: profileID})
	third, _ := st.createTask(TaskInput{Title: "Trois", ProfileID: profileID})

	if err := st.reorderTasks([]string{first.ID, second.ID, third.ID}); err != nil {
		t.Fatalf("reorderTasks: %v", err)
	}
	tasks, err := st.listTasks()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{first.ID, second.ID, third.ID}
	for i, id := range want {
		if tasks[i].ID != id {
			t.Fatalf("unexpected order at %d: %s", i, tasks[i].Title)
		}
	}

	// A reopened task comes back at the top of its profile.
	if _, err := st.setDone(second.ID, true); err != nil {
		t.Fatal(err)
	}
	reopened, err := st.setDone(second.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	tasks, err = st.listTasks()
	if err != nil {
		t.Fatal(err)
	}
	open := tasks[:3]
	if open[0].ID != reopened.ID {
		t.Fatalf("reopened task should be first, got %s", open[0].Title)
	}
}

func TestReorderSubtasks(t *testing.T) {
	st := newTestStore(t)
	task, err := st.createTask(TaskInput{Title: "Tâche"})
	if err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"Un", "Deux", "Trois"} {
		if task, err = st.addSubtask(task.ID, title); err != nil {
			t.Fatal(err)
		}
	}
	ids := []string{task.Subtasks[2].ID, task.Subtasks[0].ID, task.Subtasks[1].ID}
	updated, err := st.reorderSubtasks(task.ID, ids)
	if err != nil {
		t.Fatalf("reorderSubtasks: %v", err)
	}
	for i, id := range ids {
		if updated.Subtasks[i].ID != id || updated.Subtasks[i].Position != i {
			t.Fatalf("unexpected subtask order at %d: %+v", i, updated.Subtasks)
		}
	}
}

func TestDueReminders(t *testing.T) {
	now := time.Now().UnixMilli()
	past := now - time.Minute.Milliseconds()
	old := now - 2*reminderWindow.Milliseconds()
	future := now + time.Minute.Milliseconds()

	tasks := []Task{
		{ID: "due", DueAt: &past},
		{ID: "old", DueAt: &old},
		{ID: "future", DueAt: &future},
		{ID: "done", DueAt: &past, Done: true},
		{ID: "none"},
	}
	got := dueReminders(tasks, now, map[string]int64{})
	if len(got) != 1 || got[0].ID != "due" {
		t.Fatalf("unexpected reminders: %+v", got)
	}
	if again := dueReminders(tasks, now, map[string]int64{"due": past}); len(again) != 0 {
		t.Fatalf("an already notified task should be skipped: %+v", again)
	}
}
