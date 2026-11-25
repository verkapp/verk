package src

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Outside macOS the editing shortcuts belong to the webview: a menu
// accelerator would run the action a second time (Ctrl+V pasted twice).
func TestEditMenuShortcuts(t *testing.T) {
	roles := []application.Role{application.Undo, application.Redo, application.Cut, application.Copy, application.Paste, application.Delete, application.SelectAll}

	other := application.NewMenu()
	editMenu(other, false)
	for _, role := range roles {
		item := other.FindByRole(role)
		if item == nil {
			t.Fatalf("role %v missing", role)
		}
		if acc := item.GetAccelerator(); acc != "" {
			t.Fatalf("role %v (%s) keeps the shortcut %q outside macOS", role, item.Label(), acc)
		}
	}
	if other.FindByRole(application.Paste).Label() != "Coller" {
		t.Fatalf("paste should be labelled in French")
	}

	mac := application.NewMenu()
	editMenu(mac, true)
	if mac.FindByRole(application.Paste).GetAccelerator() == "" {
		t.Fatalf("macOS should keep Cmd+V on the paste item")
	}
}
