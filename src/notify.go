package src

import (
	"sync"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	reminderTick   = 20 * time.Second
	reminderWindow = 5 * time.Minute
)

type reminderState struct {
	mu       sync.Mutex
	notified map[string]int64
	stop     chan struct{}
}

// startNotifications initialises the native notification service and starts
// the reminder loop. Only tasks that reach their due time while Todo is
// running are announced, once per due date.
func (a *App) startNotifications() {
	if a.ctx == nil {
		return
	}
	a.reminder = &reminderState{notified: map[string]int64{}, stop: make(chan struct{})}
	if err := wruntime.InitializeNotifications(a.ctx); err != nil {
		a.notificationErr = err
		return
	}
	if settings, err := a.store.getSettings(); err == nil && settings.Notifications {
		// Ask once at startup: on macOS the first call shows the system
		// permission prompt, later calls simply return the stored answer.
		_, _ = wruntime.RequestNotificationAuthorization(a.ctx)
	}
	go a.reminderLoop()
}

func (a *App) stopNotifications() {
	if a.reminder != nil {
		select {
		case <-a.reminder.stop:
		default:
			close(a.reminder.stop)
		}
	}
	if a.ctx != nil {
		wruntime.CleanupNotifications(a.ctx)
	}
}

func (a *App) reminderLoop() {
	ticker := time.NewTicker(reminderTick)
	defer ticker.Stop()
	a.checkReminders()
	for {
		select {
		case <-a.reminder.stop:
			return
		case <-ticker.C:
			a.checkReminders()
		}
	}
}

func (a *App) checkReminders() {
	if a.store == nil || a.ctx == nil || a.reminder == nil {
		return
	}
	settings, err := a.store.getSettings()
	if err != nil || !settings.Notifications {
		return
	}
	if !wruntime.IsNotificationAvailable(a.ctx) {
		return
	}
	if allowed, err := wruntime.CheckNotificationAuthorization(a.ctx); err != nil || !allowed {
		return
	}
	tasks, err := a.store.listTasks()
	if err != nil {
		return
	}
	profiles, err := a.store.listProfiles()
	if err != nil {
		return
	}
	profileNames := make(map[string]string, len(profiles))
	for _, p := range profiles {
		profileNames[p.ID] = p.Name
	}

	a.reminder.mu.Lock()
	defer a.reminder.mu.Unlock()
	now := time.Now().UnixMilli()
	for _, task := range dueReminders(tasks, now, a.reminder.notified) {
		profile := profileNames[task.ProfileID]
		if profile == "" {
			profile = "Todo"
		}
		err := wruntime.SendNotification(a.ctx, wruntime.NotificationOptions{
			ID:       "task-" + task.ID,
			Title:    task.Title,
			Subtitle: "Rappel · " + profile,
			Body:     "Cette tâche arrive à échéance.",
		})
		if err == nil {
			a.reminder.notified[task.ID] = *task.DueAt
		}
	}
}

// dueReminders selects open tasks whose due time has just passed and that have
// not been announced for that exact due timestamp yet. The window keeps a
// long-running app from dumping every old overdue task at once.
func dueReminders(tasks []Task, now int64, notified map[string]int64) []Task {
	out := []Task{}
	cutoff := now - reminderWindow.Milliseconds()
	for _, task := range tasks {
		if task.Done || task.DueAt == nil {
			continue
		}
		due := *task.DueAt
		if due > now || due < cutoff {
			continue
		}
		if notified[task.ID] == due {
			continue
		}
		out = append(out, task)
	}
	return out
}
