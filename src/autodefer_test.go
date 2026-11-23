package src

import (
	"strconv"
	"testing"
	"time"
)

func localTime(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.Local)
}

func TestParseClock(t *testing.T) {
	for in, want := range map[string]string{"18:00": "18:00", "7:30": "07:30", " 23:59 ": "23:59", "00:00": "00:00"} {
		_, _, got, err := parseClock(in)
		if err != nil || got != want {
			t.Fatalf("parseClock(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "18", "24:00", "12:60", "12:5", "ab:cd", "1:2:3"} {
		if _, _, _, err := parseClock(in); err == nil {
			t.Fatalf("parseClock(%q) should fail", in)
		}
	}
}

func TestLastCutoffAndResumeDay(t *testing.T) {
	// Tuesday 24 November 2026.
	if got := lastCutoff(localTime(2026, 11, 24, 19, 0), 18, 0); !got.Equal(localTime(2026, 11, 24, 18, 0)) {
		t.Fatalf("after the cutoff: %v", got)
	}
	if got := lastCutoff(localTime(2026, 11, 24, 9, 0), 18, 0); !got.Equal(localTime(2026, 11, 23, 18, 0)) {
		t.Fatalf("before the cutoff: %v", got)
	}
	cases := map[int]int{24: 25, 27: 30, 28: 30, 29: 30} // Tue→Wed, Fri→Mon, Sat→Mon, Sun→Mon
	for from, to := range cases {
		if got := resumeDay(localTime(2026, 11, from, 18, 0)); !got.Equal(localTime(2026, 11, to, 0, 0)) {
			t.Fatalf("resumeDay(%d) = %v, want the %d", from, got, to)
		}
	}
}

func TestAutoDeferMoves(t *testing.T) {
	cutoff := localTime(2026, 11, 27, 18, 0) // Friday: tasks go to Monday 30
	ms := func(v time.Time) *int64 { x := v.UnixMilli(); return &x }
	tasks := []Task{
		{ID: "overdue", ProfileID: "p", DueAt: ms(localTime(2026, 11, 25, 9, 30))},
		{ID: "today-no-time", ProfileID: "p", DueAt: ms(localTime(2026, 11, 27, 23, 59))},
		{ID: "today-before", ProfileID: "p", DueAt: ms(localTime(2026, 11, 27, 17, 0))},
		{ID: "today-at-cutoff", ProfileID: "p", DueAt: ms(localTime(2026, 11, 27, 18, 0))},
		{ID: "today-evening", ProfileID: "p", DueAt: ms(localTime(2026, 11, 27, 20, 0))},
		{ID: "tomorrow", ProfileID: "p", DueAt: ms(localTime(2026, 11, 28, 10, 0))},
		{ID: "no-due", ProfileID: "p"},
		{ID: "done", ProfileID: "p", Done: true, DueAt: ms(localTime(2026, 11, 27, 9, 0))},
		{ID: "other-profile", ProfileID: "q", DueAt: ms(localTime(2026, 11, 27, 9, 0))},
	}
	moves := autoDeferMoves(tasks, "p", cutoff)
	want := map[string]time.Time{
		"overdue":         localTime(2026, 11, 30, 9, 30),
		"today-no-time":   localTime(2026, 11, 30, 23, 59),
		"today-before":    localTime(2026, 11, 30, 17, 0),
		"today-at-cutoff": localTime(2026, 11, 30, 18, 0),
	}
	if len(moves) != len(want) {
		t.Fatalf("moved %v, want %v", moves, want)
	}
	for id, due := range want {
		if moves[id] != due.UnixMilli() {
			t.Fatalf("%s moved to %v, want %v", id, time.UnixMilli(moves[id]), due)
		}
	}
}

func TestRunAutoDefer(t *testing.T) {
	st := newTestStore(t)
	app := &App{store: st}
	due := localTime(2026, 11, 24, 10, 0).UnixMilli()
	late := localTime(2026, 11, 24, 20, 0).UnixMilli()
	task, err := st.createTask(TaskInput{Title: "Envoyer le devis", DueAt: &due})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	evening, err := st.createTask(TaskInput{Title: "Appeler maman", DueAt: &late, ProfileID: task.ProfileID})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	if _, err := st.setStarted(task.ID, true); err != nil {
		t.Fatalf("setStarted: %v", err)
	}
	if _, err := st.setProfileAutoDefer(task.ProfileID, "18:00"); err != nil {
		t.Fatalf("setProfileAutoDefer: %v", err)
	}
	if _, err := st.setProfileAutoDefer(task.ProfileID, "25:00"); err == nil {
		t.Fatal("an invalid time should be refused")
	}

	// Last processed this morning; it is now 19:00, after the 18:00 cutoff.
	morning := localTime(2026, 11, 24, 9, 0).UnixMilli()
	if err := st.saveSettingValues(map[string]string{autoDeferKey(task.ProfileID): strconv.FormatInt(morning, 10)}); err != nil {
		t.Fatalf("stamp: %v", err)
	}
	now := localTime(2026, 11, 24, 19, 0)
	moved, target, err := app.runAutoDefer(now)
	if err != nil {
		t.Fatalf("runAutoDefer: %v", err)
	}
	if moved != 1 || !target.Equal(localTime(2026, 11, 25, 0, 0)) {
		t.Fatalf("moved %d to %v, want 1 to Wednesday", moved, target)
	}
	got, _ := st.getTask(task.ID)
	if got.DueAt == nil || *got.DueAt != localTime(2026, 11, 25, 10, 0).UnixMilli() {
		t.Fatalf("task not moved to Wednesday 10:00: %+v", got.DueAt)
	}
	if got.StartedAt != nil {
		t.Fatal("deferring should pause the timer")
	}
	if kept, _ := st.getTask(evening.ID); *kept.DueAt != late {
		t.Fatal("a task planned after the cutoff must stay")
	}

	// Running again for the same cutoff changes nothing.
	if moved, _, err := app.runAutoDefer(now.Add(time.Minute)); err != nil || moved != 0 {
		t.Fatalf("second run moved %d (%v), want 0", moved, err)
	}
	if dayLabel(target, now) != "demain" {
		t.Fatalf("label = %q", dayLabel(target, now))
	}
}

func TestRunAutoDeferFirstRunOnlyStamps(t *testing.T) {
	st := newTestStore(t)
	app := &App{store: st}
	due := localTime(2026, 11, 24, 10, 0).UnixMilli()
	task, err := st.createTask(TaskInput{Title: "Ranger le bureau", DueAt: &due})
	if err != nil {
		t.Fatalf("createTask: %v", err)
	}
	// The setting comes from another machine through sync: no local stamp.
	if _, err := st.db.Exec(`UPDATE profiles SET auto_defer_at = '18:00' WHERE id = ?`, task.ProfileID); err != nil {
		t.Fatalf("update profile: %v", err)
	}
	moved, _, err := app.runAutoDefer(localTime(2026, 11, 24, 19, 0))
	if err != nil || moved != 0 {
		t.Fatalf("first run moved %d (%v), want 0", moved, err)
	}
	if _, ok, _ := st.settingInt(autoDeferKey(task.ProfileID)); !ok {
		t.Fatal("first run should record its stamp")
	}
}
