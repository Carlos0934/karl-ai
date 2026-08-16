package tui

import "charm.land/bubbles/v2/key"

var (
	quitKey = key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit"))
	backKey = key.NewBinding(key.WithKeys("esc", "shift+tab"), key.WithHelp("esc/shift+tab", "back"))
)
