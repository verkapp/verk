package src

import (
	"testing"
	"time"
)

func intPtr(value int64) *int64 { return &value }

func testTask(id, title string, updatedAt int64) snapshotTask {
	return snapshotTask{
		ID:        id,
		Title:     title,
		CreatedAt: 1000,
		UpdatedAt: updatedAt,
	}
}

func testSubtask(id, taskID, title string, updatedAt int64) snapshotSubtask {
	return snapshotSubtask{ID: id, TaskID: taskID, Title: title, UpdatedAt: updatedAt}
}

func TestMergeSnapshotsUsesNewestTask(t *testing.T) {
	local := emptySnapshot()
	local.Tasks = []snapshotTask{testTask("a", "local", 10)}
	remote := emptySnapshot()
	remote.Tasks = []snapshotTask{testTask("a", "remote", 20)}

	merged := mergeSnapshots(local, remote, 100)
	if len(merged.Tasks) != 1 || merged.Tasks[0].Title != "remote" {
		t.Fatalf("remote newest should win, got %+v", merged.Tasks)
	}

	// Swapping the arguments must not change which entity is the newest.
	merged = mergeSnapshots(remote, local, 100)
	if len(merged.Tasks) != 1 || merged.Tasks[0].Title != "remote" {
		t.Fatalf("the newest entity should win regardless of side, got %+v", merged.Tasks)
	}
}

func TestMergeSnapshotsTieIsSymmetric(t *testing.T) {
	one := testTask("a", "alpha", 10)
	two := testTask("a", "zulu", 10)

	left := mergeSnapshots(snapshot{Tasks: []snapshotTask{one}}, snapshot{Tasks: []snapshotTask{two}}, 100)
	right := mergeSnapshots(snapshot{Tasks: []snapshotTask{two}}, snapshot{Tasks: []snapshotTask{one}}, 100)
	if len(left.Tasks) != 1 || left.Tasks[0].Title != right.Tasks[0].Title {
		t.Fatalf("tie-break must converge: %+v vs %+v", left.Tasks, right.Tasks)
	}
}

func TestMergeSnapshotsMergesSubtasksIndependently(t *testing.T) {
	task := testTask("t", "task", 10)
	local := emptySnapshot()
	local.Tasks = []snapshotTask{task}
	local.Subtasks = []snapshotSubtask{testSubtask("s1", "t", "local", 10)}
	remote := emptySnapshot()
	remote.Tasks = []snapshotTask{task}
	remote.Subtasks = []snapshotSubtask{testSubtask("s2", "t", "remote", 20)}

	merged := mergeSnapshots(local, remote, 100)
	if len(merged.Subtasks) != 2 {
		t.Fatalf("both subtasks should survive, got %+v", merged.Subtasks)
	}
}

func TestMergeSnapshotsAppliesTombstones(t *testing.T) {
	task := testTask("t", "task", 10)
	sub := testSubtask("s", "t", "sub", 10)
	img := snapshotImage{ID: "i", TaskID: "t", File: "i.png", UpdatedAt: 10}
	local := emptySnapshot()
	local.Tasks = []snapshotTask{task}
	local.Subtasks = []snapshotSubtask{sub}
	local.Images = []snapshotImage{img}
	remote := emptySnapshot()
	remote.Tombstones = []snapshotTombstone{
		{Kind: tombstoneTask, ID: "t", DeletedAt: 20},
		{Kind: tombstoneSubtask, ID: "s", DeletedAt: 20},
		{Kind: tombstoneImage, ID: "i", DeletedAt: 20},
	}

	merged := mergeSnapshots(local, remote, 100)
	if len(merged.Tasks) != 0 || len(merged.Subtasks) != 0 || len(merged.Images) != 0 {
		t.Fatalf("tombstones should remove the entities, got %+v", merged)
	}
	if len(merged.Tombstones) != 3 {
		t.Fatalf("tombstones should be preserved, got %+v", merged.Tombstones)
	}
}

func TestMergeSnapshotsResurrectsNewerEdit(t *testing.T) {
	local := emptySnapshot()
	local.Tasks = []snapshotTask{testTask("t", "edited later", 30)}
	remote := emptySnapshot()
	remote.Tombstones = []snapshotTombstone{{Kind: tombstoneTask, ID: "t", DeletedAt: 20}}

	merged := mergeSnapshots(local, remote, 100)
	if len(merged.Tasks) != 1 {
		t.Fatalf("an edit newer than the deletion should win, got %+v", merged)
	}
}

func TestMergeSnapshotsPrunesOldTombstones(t *testing.T) {
	now := time.Now().UnixMilli()
	remote := emptySnapshot()
	remote.Tombstones = []snapshotTombstone{{Kind: tombstoneTask, ID: "old", DeletedAt: now - tombstoneRetention.Milliseconds() - 1}}

	merged := mergeSnapshots(emptySnapshot(), remote, now)
	if len(merged.Tombstones) != 0 {
		t.Fatalf("old tombstones should be pruned, got %+v", merged.Tombstones)
	}
}

func TestStoreSnapshotAndReplaceRoundTrip(t *testing.T) {
	st := newTestStore(t)
	task, err := st.createTask(TaskInput{Title: "Acheter du pain", Description: "au marché", DueAt: intPtr(2000)})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	if _, err := st.addSubtask(task.ID, "Prendre de la monnaie"); err != nil {
		t.Fatalf("addSubtask: %v", err)
	}

	snap, err := st.snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snap.Tasks) != 1 || len(snap.Subtasks) != 1 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
	if snap.Tasks[0].UpdatedAt == 0 || snap.Subtasks[0].UpdatedAt == 0 {
		t.Fatalf("updated_at should be set, got %+v", snap)
	}

	other := newTestStore(t)
	if err := other.replaceSnapshot(snap); err != nil {
		t.Fatalf("replaceSnapshot: %v", err)
	}
	restored, err := other.snapshot()
	if err != nil {
		t.Fatalf("snapshot after replace: %v", err)
	}
	if !snapshotsEqual(snap, restored) {
		t.Fatalf("round trip changed the data:\n%+v\n%+v", snap, restored)
	}
}

func TestStoreMergeSnapshotUpdatesDatabase(t *testing.T) {
	st := newTestStore(t)
	task, err := st.createTask(TaskInput{Title: "Locale"})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	remote := emptySnapshot()
	remote.Tasks = []snapshotTask{
		{ID: task.ID, Title: "Distante", CreatedAt: task.CreatedAt, UpdatedAt: time.Now().UnixMilli() + 1000},
		testTask("new", "Nouvelle", time.Now().UnixMilli()),
	}
	_, changed, err := st.mergeSnapshot(remote)
	if err != nil {
		t.Fatalf("mergeSnapshot: %v", err)
	}
	if !changed {
		t.Fatal("merge should report a change")
	}
	tasks, err := st.listTasks()
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %+v", tasks)
	}
}

func TestDeleteRecordsTombstones(t *testing.T) {
	st := newTestStore(t)
	task, err := st.createTask(TaskInput{Title: "À supprimer"})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	updated, err := st.addSubtask(task.ID, "Enfant")
	if err != nil {
		t.Fatalf("addSubtask: %v", err)
	}
	if err := st.deleteTask(updated.ID); err != nil {
		t.Fatalf("deleteTask: %v", err)
	}
	snap, err := st.snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snap.Tasks) != 0 {
		t.Fatalf("task should be gone, got %+v", snap.Tasks)
	}
	found := false
	for _, tomb := range snap.Tombstones {
		if tomb.Kind == tombstoneTask && tomb.ID == task.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing task tombstone: %+v", snap.Tombstones)
	}
}

func TestMergeSnapshotKeepsLocalRunningTimer(t *testing.T) {
	st := newTestStore(t)
	task, err := st.createTask(TaskInput{Title: "En cours"})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	started, err := st.setStarted(task.ID, true)
	if err != nil {
		t.Fatalf("setStarted: %v", err)
	}
	if started.StartedAt == nil {
		t.Fatal("timer should be running")
	}

	remote := emptySnapshot()
	remote.Tasks = []snapshotTask{
		{ID: task.ID, Title: "En cours", CreatedAt: task.CreatedAt, UpdatedAt: time.Now().UnixMilli() + 1000},
	}
	merged, _, err := st.mergeSnapshot(remote)
	if err != nil {
		t.Fatalf("mergeSnapshot: %v", err)
	}
	if len(merged.Tasks) != 1 || merged.Tasks[0].StartedAt == nil {
		t.Fatalf("local running timer should be preserved, got %+v", merged.Tasks)
	}

	remote.Tasks[0].StartedAt = intPtr(42)
	merged, _, err = st.mergeSnapshot(remote)
	if err != nil {
		t.Fatalf("mergeSnapshot: %v", err)
	}
	if len(merged.Tasks) != 1 || merged.Tasks[0].StartedAt == nil || *merged.Tasks[0].StartedAt == 42 {
		t.Fatalf("a remote started_at must not start a phantom timer, got %+v", merged.Tasks)
	}
}
