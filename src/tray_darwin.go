package src

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>

void todoTrayStart(const void *png, int length);
void todoTrayStop(void);
void todoActivateApp(void);
*/
import "C"

import (
	_ "embed"
	"sync/atomic"
	"unsafe"
)

// A native NSStatusItem is used instead of a systray library: those replace
// the NSApplication delegate, which belongs to Wails.

//go:embed trayicons/template.png
var trayTemplateIcon []byte

var trayOwner atomic.Pointer[App]

func (a *App) startTray() {
	trayOwner.Store(a)
	C.todoTrayStart(unsafe.Pointer(&trayTemplateIcon[0]), C.int(len(trayTemplateIcon)))
}

func (a *App) stopTray() {
	C.todoTrayStop()
}

func activateApp() {
	C.todoActivateApp()
}

//export todoTrayClicked
func todoTrayClicked(id C.int) {
	app := trayOwner.Load()
	if app == nil {
		return
	}
	// Called on the AppKit main thread: leave it before calling the runtime.
	go app.trayAction(int(id))
}
