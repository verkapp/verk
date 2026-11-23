package src

import (
	"context"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
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
//
// The notification service is started by hand rather than registered with
// Wails: a failing service would abort the whole application (for example
// a macOS build without a bundle identifier), while Todo only needs to
// report that reminders are unavailable.
func (a *App) startNotifications(ctx context.Context) {
	a.reminder = &reminderState{notified: map[string]int64{}, stop: make(chan struct{})}
	notifier := notifications.New()
	if err := notifier.ServiceStartup(ctx, application.ServiceOptions{}); err != nil {
		a.notificationErr = err
		return
	}
	a.notifier = notifier
	if settings, err := a.store.getSettings(); err == nil && settings.Notifications {
		// Ask once at startup: on macOS the first call shows the system
		// permission prompt, later calls simply return the stored answer.
		_, _ = notifier.RequestNotificationAuthorization()
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
	if a.notifier != nil {
		_ = a.notifier.ServiceShutdown()
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
	if a.store == nil || a.notifier == nil || a.reminder == nil {
		return
	}
	settings, err := a.store.getSettings()
	if err != nil || !settings.Notifications {
		return
	}
	if allowed, err := a.notifier.CheckNotificationAuthorization(); err != nil || !allowed {
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
		err := a.notifier.SendNotification(notifications.NotificationOptions{
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
