GioRefreshRate
--------

Allows to set the display refresh rate in [Gio](https://gioui.org) apps on Android. Some manufacturers limit apps using SurfaceView to 60hz, this library can bring it back and have that scroll be smooth.

<img src="pic.jpg" width="480">

## Getting Started

Run `go get github.com/ilidemi/giorefreshrate`

Call `giorefreshrate.PreferHighRefreshRate()` or `giorefreshrate.PreferLowRefreshRate()` before the event loop and provide `giorefreshrate` access to the Window events:

```diff
+   giorefreshrate.PreferHighRefreshRate()

    for e := range w.Events() { // Gio main event loop
+       giorefreshrate.ListenEvents(e, w)

        switch e := e.(type) {
            // ...
        }
    }
```

That's it! The refresh rate will be the highest or lowest supported for the current display resolution.

## Notes

Uses the same approach as [flutter_displaymode](https://github.com/ajinasokan/flutter_displaymode).
