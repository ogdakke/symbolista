package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func RunTUI(directory string, showPercentages bool, workerCount int, includeDotfiles bool, asciiOnly bool) error {
	model := NewModel(directory, showPercentages, workerCount, includeDotfiles, asciiOnly)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}
