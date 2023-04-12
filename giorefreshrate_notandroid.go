//go:build !android

package giorefreshrate

import (
	"gioui.org/app"
	"gioui.org/io/event"
)

func listenEvents(event event.Event, w *app.Window) error {
	return nil
}
