package tui

import (
	"fmt"
	"sort"
	"strconv"
	"unicode"

	"github.com/NimbleMarkets/ntcharts/barchart"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ogdakke/symbolista/internal/counter"
	"github.com/ogdakke/symbolista/internal/gitignore"
	"github.com/ogdakke/symbolista/internal/logger"
	"github.com/ogdakke/symbolista/internal/traversal"
)

type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterLettersNumbers
	FilterSymbols
)

type Model struct {
	directory       string
	showPercentages bool
	workerCount     int
	includeDotfiles bool
	asciiOnly       bool

	charCounts        counter.CharCounts
	filteredCounts    counter.CharCounts
	chart             barchart.Model
	ready             bool
	loading           bool
	err               error
	filterMode        FilterMode
	excludeWhitespace bool

	width  int
	height int
}

type analysisCompleteMsg struct {
	counts counter.CharCounts
	err    error
}

type analysisStartMsg struct{}

func isLetterOrNumber(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isSymbol(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || unicode.IsSpace(r)
}

func (m *Model) applyFilter() {
	m.filteredCounts = m.filteredCounts[:0]

	for _, charCount := range m.charCounts {
		if len(charCount.Char) == 0 {
			continue
		}

		r := []rune(charCount.Char)[0]

		// Skip whitespace characters if exclusion is enabled
		if m.excludeWhitespace && isWhitespace(r) {
			continue
		}

		switch m.filterMode {
		case FilterAll:
			m.filteredCounts = append(m.filteredCounts, charCount)
		case FilterLettersNumbers:
			if isLetterOrNumber(r) {
				m.filteredCounts = append(m.filteredCounts, charCount)
			}
		case FilterSymbols:
			if isSymbol(r) {
				m.filteredCounts = append(m.filteredCounts, charCount)
			}
		}
	}

	// Re-sort the filtered counts
	sort.Sort(m.filteredCounts)
}

func (m FilterMode) String() string {
	switch m {
	case FilterAll:
		return "All"
	case FilterLettersNumbers:
		return "Letters & Numbers"
	case FilterSymbols:
		return "Symbols"
	default:
		return "All"
	}
}

func NewModel(directory string, showPercentages bool, workerCount int, includeDotfiles bool, asciiOnly bool) Model {
	return Model{
		directory:         directory,
		showPercentages:   showPercentages,
		workerCount:       workerCount,
		includeDotfiles:   includeDotfiles,
		asciiOnly:         asciiOnly,
		loading:           true,
		filterMode:        FilterAll,
		excludeWhitespace: true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		startAnalysis(m.directory, m.workerCount, m.includeDotfiles, m.asciiOnly),
		tea.EnterAltScreen,
	)
}

func startAnalysis(directory string, workerCount int, includeDotfiles bool, asciiOnly bool) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		logger.Info("Starting TUI analysis", "directory", directory)

		matcher, err := gitignore.NewMatcher(directory, includeDotfiles)
		if err != nil {
			return analysisCompleteMsg{err: err}
		}

		result, err := traversal.WalkDirectoryConcurrent(directory, matcher, workerCount, asciiOnly)
		if err != nil {
			return analysisCompleteMsg{err: err}
		}

		charMap := result.CharMap
		totalChars := result.TotalChars

		var counts counter.CharCounts
		for char, count := range charMap {
			percentage := float64(count) / float64(totalChars) * 100
			counts = append(counts, counter.CharCount{
				Char:       string(char),
				Count:      count,
				Percentage: percentage,
			})
		}

		sort.Sort(counts)

		return analysisCompleteMsg{counts: counts}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.ready {
			m.applyFilter()
			m.updateChart()
		}
		return m, nil

	case analysisCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.charCounts = msg.counts
		m.ready = true
		m.applyFilter()
		m.updateChart()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			if m.ready {
				m.loading = true
				m.ready = false
				return m, startAnalysis(m.directory, m.workerCount, m.includeDotfiles, m.asciiOnly)
			}
		case "a":
			if m.ready {
				m.filterMode = FilterAll
				m.applyFilter()
				m.updateChart()
			}
		case "l":
			if m.ready {
				m.filterMode = FilterLettersNumbers
				m.applyFilter()
				m.updateChart()
			}
		case "s":
			if m.ready {
				m.filterMode = FilterSymbols
				m.applyFilter()
				m.updateChart()
			}
		case "w":
			if m.ready {
				m.excludeWhitespace = !m.excludeWhitespace
				m.applyFilter()
				m.updateChart()
			}
		}
	}

	return m, nil
}

func (m *Model) updateChart() {
	if !m.ready || len(m.filteredCounts) == 0 {
		return
	}

	chartWidth := m.width - 2
	chartHeight := m.height - 8

	if chartWidth < 30 {
		chartWidth = 30
	}
	if chartHeight < 10 {
		chartHeight = 10
	}

	m.chart = barchart.New(chartWidth, chartHeight)

	// Calculate how many items can fit based on average label width
	// Each bar with label needs roughly 11 characters of space
	estimatedLabelWidth := 11
	maxItems := min(chartWidth/estimatedLabelWidth, len(m.filteredCounts), 25)

	var barData []barchart.BarData
	colors := []string{"10", "9", "11", "14", "13", "12", "6", "5", "4", "3", "2", "1"}

	for i := range maxItems {
		char := m.filteredCounts[i]
		displayChar := char.Char

		switch char.Char {
		case " ":
			displayChar = "⎵" // Unicode space symbol
		case "\t":
			displayChar = "⇥" // Unicode tab symbol
		case "\n":
			displayChar = "↵" // Unicode return/newline symbol
		case "\r":
			displayChar = "⏎" // Unicode carriage return symbol
		}

		color := colors[i%len(colors)]

		// Format the count for display - use shorter format for large numbers
		countStr := strconv.Itoa(char.Count)
		if char.Count >= 1000 {
			countStr = fmt.Sprintf("%.1fk", float64(char.Count)/1000)
		}

		// Put the count in the same line as the character with a separator
		labelWithCount := fmt.Sprintf("%s:%s", displayChar, countStr)

		barData = append(barData, barchart.BarData{
			Label: labelWithCount,
			Values: []barchart.BarValue{
				{Name: strconv.Itoa(char.Count), Value: float64(char.Count), Style: lipgloss.NewStyle().Foreground(lipgloss.Color(color))},
			},
		})
	}

	// Configure chart to show y-axis with count values
	m.chart.SetShowAxis(true)
	m.chart.PushAll(barData)
	m.chart.Draw()

}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit", m.err)
	}

	if m.loading {
		return "Analyzing files...\n\nPress 'q' to quit"
	}

	if !m.ready {
		return "Loading...\n\nPress 'q' to quit"
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14")).
		Render("Symbol Distribution")

	whitespaceStatus := ""
	if m.excludeWhitespace {
		whitespaceStatus = " | No whitespace"
	}

	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(fmt.Sprintf("Directory: %s | Filter: %s%s | Showing: %d/%d chars", m.directory, m.filterMode.String(), whitespaceStatus, len(m.filteredCounts), len(m.charCounts)))

	chart := m.chart.View()

	// Create a legend showing top characters with their counts
	legend := m.createLegend()

	controls := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Controls: 'a' all | 'l' letters/numbers | 's' symbols | 'w' toggle whitespace | 'r' refresh | 'q' quit")

	return fmt.Sprintf("%s\n%s\n\n%s\n\n%s\n\n%s", title, info, chart, legend, controls)
}

func (m Model) createLegend() string {
	if !m.ready || len(m.filteredCounts) == 0 {
		return ""
	}

	maxItems := min(m.width/12, len(m.filteredCounts), 32) // Fit legend items on screen
	colors := []string{"10", "9", "11", "14", "13", "12", "6", "5", "4", "3", "2", "1"}

	var legendItems []string
	for i := range maxItems {
		char := m.filteredCounts[i]
		displayChar := char.Char

		switch char.Char {
		case " ":
			displayChar = "⎵" // Unicode space symbol
		case "\t":
			displayChar = "⇥" // Unicode tab symbol
		case "\n":
			displayChar = "↵" // Unicode return/newline symbol
		case "\r":
			displayChar = "⏎" // Unicode carriage return symbol
		}

		color := colors[i%len(colors)]
		coloredChar := lipgloss.NewStyle().
			Foreground(lipgloss.Color(color)).
			Bold(true).
			Render(displayChar)

		percentage := ""
		if m.showPercentages {
			percentage = fmt.Sprintf(" (%.1f%%)", char.Percentage)
		}

		// Since counts are now shown on bars, make legend more compact
		legendItems = append(legendItems, fmt.Sprintf("%s%s", coloredChar, percentage))
	}

	legendTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Top Characters: ")

	// Join items with spaces, making sure they fit on one line
	legendContent := ""
	for i, item := range legendItems {
		if i > 0 {
			legendContent += " | "
		}
		legendContent += item
	}

	return legendTitle + legendContent
}
