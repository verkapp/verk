package src

import (
	"errors"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// notificationsReady reports why reminders cannot be shown, if they cannot.
func (a *App) notificationsReady() error {
	if a.notificationErr != nil {
		return a.notificationErr
	}
	if a.notifier == nil {
		return errors.New("les notifications ne sont pas encore initialisées")
	}
	return nil
}

func (a *App) NotificationStatus() (NotificationStatus, error) {
	if err := a.ready(); err != nil {
		return NotificationStatus{}, err
	}
	if err := a.notificationsReady(); err != nil {
		return NotificationStatus{Error: err.Error()}, nil
	}
	status := NotificationStatus{Available: true}
	authorized, err := a.notifier.CheckNotificationAuthorization()
	if err != nil {
		status.Error = err.Error()
		return status, nil
	}
	status.Authorized = authorized
	return status, nil
}

// RequestNotifications asks the operating system for permission. On macOS this
// shows the system prompt the first time.
func (a *App) RequestNotifications() (NotificationStatus, error) {
	if err := a.ready(); err != nil {
		return NotificationStatus{}, err
	}
	if err := a.notificationsReady(); err != nil {
		return NotificationStatus{Error: err.Error()}, nil
	}
	authorized, err := a.notifier.RequestNotificationAuthorization()
	if err != nil {
		return NotificationStatus{Available: true, Error: err.Error()}, nil
	}
	return NotificationStatus{Available: true, Authorized: authorized}, nil
}

func (a *App) TestNotification() error {
	if err := a.ready(); err != nil {
		return err
	}
	if err := a.notificationsReady(); err != nil {
		return err
	}
	if allowed, err := a.notifier.CheckNotificationAuthorization(); err != nil {
		return err
	} else if !allowed {
		if allowed, err = a.notifier.RequestNotificationAuthorization(); err != nil {
			return err
		} else if !allowed {
			return errors.New("les notifications sont refusées dans les réglages du système")
		}
	}
	return a.notifier.SendNotification(notifications.NotificationOptions{
		ID:       "test",
		Title:    "Todo",
		Subtitle: "Les rappels sont prêts",
		Body:     "Tu recevras une notification quand une tâche arrive à échéance.",
	})
}
