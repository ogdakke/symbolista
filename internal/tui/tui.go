package tui

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ogdakke/symbolista/internal/counter"
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

func RunTUIFromJSON(jsonFile string) error {
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	var jsonOutput counter.JSONOutput
	if err := json.Unmarshal(data, &jsonOutput); err != nil {
		return fmt.Errorf("failed to parse JSON file: %w", err)
	}

	model := NewModelFromJSON(jsonOutput)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err = p.Run()
	return err
}
