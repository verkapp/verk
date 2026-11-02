package src

import (
	"errors"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) NotificationStatus() (NotificationStatus, error) {
	if err := a.ready(); err != nil {
		return NotificationStatus{}, err
	}
	status := NotificationStatus{}
	if a.ctx == nil {
		status.Error = "les notifications ne sont pas encore initialisées"
		return status, nil
	}
	if a.notificationErr != nil {
		status.Error = a.notificationErr.Error()
		return status, nil
	}
	status.Available = wruntime.IsNotificationAvailable(a.ctx)
	authorized, err := wruntime.CheckNotificationAuthorization(a.ctx)
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
	if a.ctx == nil {
		return NotificationStatus{Error: "les notifications ne sont pas encore initialisées"}, nil
	}
	if a.notificationErr != nil {
		return NotificationStatus{Error: a.notificationErr.Error()}, nil
	}
	authorized, err := wruntime.RequestNotificationAuthorization(a.ctx)
	if err != nil {
		return NotificationStatus{Available: wruntime.IsNotificationAvailable(a.ctx), Error: err.Error()}, nil
	}
	return NotificationStatus{
		Available:  wruntime.IsNotificationAvailable(a.ctx),
		Authorized: authorized,
	}, nil
}

func (a *App) TestNotification() error {
	if err := a.ready(); err != nil {
		return err
	}
	if a.ctx == nil {
		return errors.New("les notifications ne sont pas encore initialisées")
	}
	if a.notificationErr != nil {
		return a.notificationErr
	}
	if !wruntime.IsNotificationAvailable(a.ctx) {
		return errors.New("les notifications ne sont pas disponibles sur ce système")
	}
	if allowed, err := wruntime.CheckNotificationAuthorization(a.ctx); err != nil {
		return err
	} else if !allowed {
		if allowed, err = wruntime.RequestNotificationAuthorization(a.ctx); err != nil {
			return err
		} else if !allowed {
			return errors.New("les notifications sont refusées dans les réglages du système")
		}
	}
	return wruntime.SendNotification(a.ctx, wruntime.NotificationOptions{
		ID:       "test",
		Title:    "Todo",
		Subtitle: "Les rappels sont prêts",
		Body:     "Tu recevras une notification quand une tâche arrive à échéance.",
	})
}
