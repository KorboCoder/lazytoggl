package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Space   key.Binding
	Create  key.Binding
	Log     key.Binding
	Edit    key.Binding
	Delete  key.Binding
	Refresh key.Binding
	Tab     key.Binding
	Help    key.Binding
	Quit    key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "resume/stop")),
		Space:   key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "pause/resume")),
		Create:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "new entry")),
		Log:     key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "log")), // Toggl's manual-mode key
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch panel")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Space, k.Create, k.Log, k.Enter, k.Edit, k.Delete, k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Tab},
		{k.Space, k.Create, k.Log, k.Enter},
		{k.Edit, k.Delete, k.Refresh},
		{k.Help, k.Quit},
	}
}
