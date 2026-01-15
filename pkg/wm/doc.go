/*
Package wm provides window management functionality for the WebOS operating system.

This package implements the backend window management capabilities, including:
  - Window creation and lifecycle management
  - Window state tracking (minimized, maximized, focused)
  - Virtual desktop support
  - Window snapping and layout algorithms

The window manager coordinates with the frontend JavaScript implementation
to provide a complete multi-window GUI experience in the browser.

Example usage:

	manager := wm.NewManager()
	windowID, err := manager.CreateWindow("My Window", 100, 100, 800, 600)
	if err != nil {
		// handle error
	}
	err = manager.MaximizeWindow(windowID)
	if err != nil {
		// handle error
	}
*/
package wm
