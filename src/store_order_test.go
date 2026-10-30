package src

import (
	"testing"
	"time"
)

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
