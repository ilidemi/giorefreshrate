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

func PreferHighRefreshRate() {
	preference = refreshRateHigh
}

func PreferLowRefreshRate() {
	preference = refreshRateLow
}

func ListenEvents(event event.Event, w *app.Window) error {
	return listenEvents(event, w)
}
