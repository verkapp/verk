package src

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func runTestCommand(t *testing.T, name string, args ...string) {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func testApp(st *store) *App {
	return &App{store: st, gitAvailable: true}
}

func TestSyncConvergesBetweenTwoMachines(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	runTestCommand(t, "git", "init", "--bare", "-b", "main", remote)

	st1 := newTestStore(t)
	st2 := newTestStore(t)
	a1 := testApp(st1)
	a2 := testApp(st2)
	cfg := syncConfig{Remote: remote, Enabled: true, Branch: "main"}
	sync := func(a *App) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := a.performSync(ctx, cfg); err != nil {
			t.Fatalf("performSync: %v", err)
		}
	}

	first, err := st1.createTask(TaskInput{Title: "Depuis la machine 1"})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	sync(a1)
	sync(a2)

	tasks, err := st2.listTasks()
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Depuis la machine 1" {
		t.Fatalf("machine 2 should receive the task, got %+v", tasks)
	}

	second, err := st2.createTask(TaskInput{Title: "Depuis la machine 2"})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	sync(a2)
	sync(a1)

	if _, err := st1.updateTask(first.ID, TaskInput{Title: "Depuis la machine 1, modifiée"}); err != nil {
		t.Fatalf("updateTask: %v", err)
	}
	if _, err := st1.addSubtask(first.ID, "Sous-tâche de la machine 1"); err != nil {
		t.Fatalf("addSubtask: %v", err)
	}
	if _, err := st2.addSubtask(first.ID, "Sous-tâche de la machine 2"); err != nil {
		t.Fatalf("addSubtask: %v", err)
	}
	sync(a1)
	sync(a2)
	sync(a1)

	for _, st := range []*store{st1, st2} {
		tasks, err = st.listTasks()
		if err != nil {
			t.Fatalf("listTasks: %v", err)
		}
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %+v", tasks)
		}
		var titles []string
		var subtasks int
		for _, task := range tasks {
			titles = append(titles, task.Title)
			subtasks += len(task.Subtasks)
		}
		if subtasks != 2 {
			t.Fatalf("both machines should have the two subtasks, got %d in %v", subtasks, titles)
		}
	}

	if err := st1.deleteTask(second.ID); err != nil {
		t.Fatalf("deleteTask: %v", err)
	}
	sync(a1)
	sync(a2)

	tasks, err = st2.listTasks()
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != first.ID {
		t.Fatalf("deletion should propagate, got %+v", tasks)
	}
}

func TestConfigureSyncStoresRemoteAndSyncs(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	runTestCommand(t, "git", "init", "--bare", "-b", "main", remote)

	st1 := newTestStore(t)
	st2 := newTestStore(t)
	a1 := testApp(st1)
	a2 := testApp(st2)
	if _, err := st1.createTask(TaskInput{Title: "Configurée"}); err != nil {
		t.Fatalf("createTask: %v", err)
	}

	if err := a1.configureSync(remote, true); err != nil {
		t.Fatalf("configureSync: %v", err)
	}
	cfg, err := st1.syncConfig()
	if err != nil {
		t.Fatalf("syncConfig: %v", err)
	}
	if cfg.Remote != remote || !cfg.Enabled || cfg.Branch != "main" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	a1.stopSyncTimer()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if _, err := a1.syncNow(ctx, true); err != nil {
		t.Fatalf("syncNow: %v", err)
	}
	status := a1.currentSyncStatus()
	if !status.Configured || status.LastSyncAt == 0 || status.LastError != "" {
		t.Fatalf("unexpected status: %+v", status)
	}

	if err := a2.configureSync(remote, true); err != nil {
		t.Fatalf("configureSync on machine 2: %v", err)
	}
	a2.stopSyncTimer()
	if _, err := a2.syncNow(ctx, true); err != nil {
		t.Fatalf("syncNow on machine 2: %v", err)
	}
	tasks, err := st2.listTasks()
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Configurée" {
		t.Fatalf("machine 2 should receive the task, got %+v", tasks)
	}

	if err := a1.configureSync("", false); err != nil {
		t.Fatalf("disable sync: %v", err)
	}
	cfg, err = st1.syncConfig()
	if err != nil {
		t.Fatalf("syncConfig after disable: %v", err)
	}
	if cfg.Remote != "" || cfg.Enabled {
		t.Fatalf("sync should be disabled, got %+v", cfg)
	}
}

func TestSyncMovesImageFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	runTestCommand(t, "git", "init", "--bare", "-b", "main", remote)

	st1 := newTestStore(t)
	st2 := newTestStore(t)
	a1 := testApp(st1)
	a2 := testApp(st2)
	cfg := syncConfig{Remote: remote, Enabled: true, Branch: "main"}
	sync := func(a *App) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := a.performSync(ctx, cfg); err != nil {
			t.Fatalf("performSync: %v", err)
		}
	}

	task, err := st1.createTask(TaskInput{Title: "Avec une image"})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	// A one-pixel PNG, enough for the import validation.
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	source := filepath.Join(t.TempDir(), "pixel.png")
	if err := os.WriteFile(source, png, 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}
	if _, err := st1.importAttachment(task.ID, source); err != nil {
		t.Fatalf("importAttachment: %v", err)
	}

	sync(a1)
	sync(a2)

	tasks, err := st2.listTasks()
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	if len(tasks) != 1 || len(tasks[0].Images) != 1 {
		t.Fatalf("machine 2 should receive the image metadata, got %+v", tasks)
	}
	if tasks[0].Images[0].DataURL == "" {
		t.Fatal("the image file was not copied to machine 2")
	}
}

func TestSyncToEmptyRemoteCreatesBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	runTestCommand(t, "git", "init", "--bare", "-b", "main", remote)

	st := newTestStore(t)
	app := testApp(st)
	if _, err := st.createTask(TaskInput{Title: "Première"}); err != nil {
		t.Fatalf("createTask: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if _, err := app.performSync(ctx, syncConfig{Remote: remote, Enabled: true, Branch: "main"}); err != nil {
		t.Fatalf("performSync: %v", err)
	}
	out, err := exec.Command("git", "--git-dir", remote, "log", "--oneline", "main").CombinedOutput()
	if err != nil {
		t.Fatalf("remote has no main branch: %v\n%s", err, out)
	}
}
