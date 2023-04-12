package giorefreshrate

import (
	"gioui.org/app"
	"gioui.org/io/event"
)

type refreshRatePreference int

const (
	refreshRateNone = iota
	refreshRateHigh
	refreshRateLow
)

var preference refreshRatePreference

// Prefer the highest refresh rate supported by the display at the current resolution. Needs to be called
// before any window events are consumed. The refresh rate will be applied by ListenEvents() when it sees
// that the window is ready.
func PreferHighRefreshRate() {
	preference = refreshRateHigh
}

// Prefer the lowest refresh rate supported by the display at the current resolution. Needs to be called
// before any window events are consumed. The refresh rate will be applied by ListenEvents() when it sees
// that the window is ready.
func PreferLowRefreshRate() {
	preference = refreshRateLow
}

// ListenEvents must get all the events from Gio, in order to get the GioView once it's ready and apply the
// refresh rate to the window. It needs to be called from the main UI loop.
//
// Example:
//
//	select {
//	case e := <-w.Events():
//		giorefreshrate.ListenEvents(e, w)
//
//		switch e := e.(type) {
//	     (( ... your code ...  ))
func ListenEvents(event event.Event, w *app.Window) error {
	return listenEvents(event, w)
}
