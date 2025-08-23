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
	"github.com/ogdakke/symbolista/internal/logger"
)

type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterLettersNumbers
	FilterSymbols
)

type LabelMode int

const (
	LabelCount LabelMode = iota
	LabelPercentage
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

	// Horizontal scrolling
	scrollOffset int
	maxVisible   int

	// Label display mode
	labelMode LabelMode

	// File statistics and timing
	result counter.AnalysisResult

	// Progress tracking
	filesFound     int
	filesProcessed int
	progressChan   chan progressMsg
}

type analysisCompleteMsg struct {
	result counter.AnalysisResult
	err    error
}

type progressMsg struct {
	filesFound     int
	filesProcessed int
}

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

	sort.Sort(m.filteredCounts)

	m.scrollOffset = 0
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

func (m LabelMode) String() string {
	switch m {
	case LabelCount:
		return "Count"
	case LabelPercentage:
		return "Percentage"
	default:
		return "Count"
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

type analysisStartedMsg struct {
	progressChan chan progressMsg
	doneChan     chan analysisCompleteMsg
}

func listenForProgress(progressChan <-chan progressMsg) tea.Cmd {
	return func() tea.Msg {
		progress, ok := <-progressChan
		if !ok {
			return nil
		}
		return progress
	}
}

func listenForCompletion(doneChan <-chan analysisCompleteMsg) tea.Cmd {
	return func() tea.Msg {
		return <-doneChan
	}
}

func startAnalysis(directory string, workerCount int, includeDotfiles bool, asciiOnly bool) tea.Cmd {
	return func() tea.Msg {
		logger.Info("Starting async TUI analysis", "directory", directory)

		progressChan := make(chan progressMsg, 10)
		doneChan := make(chan analysisCompleteMsg, 1)

		go func() {
			defer close(progressChan)
			defer close(doneChan)

			progressFunc := func(filesFound, filesProcessed int) {
				select {
				case progressChan <- progressMsg{
					filesFound:     filesFound,
					filesProcessed: filesProcessed,
				}:
				default:
					// Channel full, skip update
				}
			}

			result, err := counter.AnalyzeSymbols(directory, workerCount, includeDotfiles, asciiOnly, progressFunc)

			doneChan <- analysisCompleteMsg{
				result: result,
				err:    err,
			}
		}()

		return analysisStartedMsg{
			progressChan: progressChan,
			doneChan:     doneChan,
		}
	}
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

	case analysisStartedMsg:

		m.progressChan = msg.progressChan
		return m, tea.Batch(
			listenForProgress(msg.progressChan),
			listenForCompletion(msg.doneChan),
		)

	case progressMsg:
		if m.loading && msg.filesFound > 0 {
			m.filesFound = msg.filesFound
			m.filesProcessed = msg.filesProcessed

			return m, listenForProgress(m.progressChan)
		}
		return m, nil

	case analysisCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.result = msg.result
		m.charCounts = msg.result.CharCounts
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
		case "left":
			if m.ready && m.scrollOffset > 0 {
				m.scrollOffset--
				m.updateChart()
			}
		case "right":
			if m.ready && m.scrollOffset < len(m.filteredCounts)-m.maxVisible {
				m.scrollOffset++
				m.updateChart()
			}
		case "home":
			if m.ready {
				m.scrollOffset = 0
				m.updateChart()
			}
		case "end":
			if m.ready && len(m.filteredCounts) > m.maxVisible {
				m.scrollOffset = len(m.filteredCounts) - m.maxVisible
				m.updateChart()
			}
		case "t":
			if m.ready {
				m.labelMode = (m.labelMode + 1) % 2
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

	// Account for border (2 chars) and padding (4 chars: 2 left + 2 right) and some margin
	chartWidth := m.width - 7
	chartHeight := m.height - 10

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
	m.maxVisible = min(chartWidth/estimatedLabelWidth, len(m.filteredCounts), 25)

	// Ensure scroll offset is within bounds
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	maxScrollOffset := max(0, len(m.filteredCounts)-m.maxVisible)
	if m.scrollOffset > maxScrollOffset {
		m.scrollOffset = maxScrollOffset
	}

	var barData []barchart.BarData
	colors := []string{"10", "9", "11", "14", "13", "12", "6", "5", "4", "3", "2", "1"}

	// Calculate visible range
	startIndex := m.scrollOffset
	endIndex := min(startIndex+m.maxVisible, len(m.filteredCounts))

	for i := startIndex; i < endIndex; i++ {
		char := m.filteredCounts[i]
		displayChar := char.Char

		switch char.Char {
		case " ":
			displayChar = "⎵"
		case "\t":
			displayChar = "⇥"
		case "\n":
			displayChar = "↵"
		case "\r":
			displayChar = "⏎"
		}

		// Use original index for color consistency across scrolling
		color := colors[i%len(colors)]

		// Format the value based on label mode
		var valueStr string
		switch m.labelMode {
		case LabelCount:
			if char.Count >= 1000000 {
				valueStr = fmt.Sprintf("%.1fM", float64(char.Count)/1000000)
			} else if char.Count >= 1000 {
				valueStr = fmt.Sprintf("%.1fk", float64(char.Count)/1000)
			} else {
				valueStr = strconv.Itoa(char.Count)
			}
		case LabelPercentage:
			valueStr = fmt.Sprintf("%.1f%%", char.Percentage)
		}

		labelWithCount := fmt.Sprintf("%s:%s", displayChar, valueStr)

		barData = append(barData, barchart.BarData{
			Label: labelWithCount,
			Values: []barchart.BarValue{
				{Name: strconv.Itoa(char.Count), Value: float64(char.Count), Style: lipgloss.NewStyle().Foreground(lipgloss.Color(color))},
			},
		})
	}

	m.chart.PushAll(barData)
	m.chart.Draw()

}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit", m.err)
	}

	if m.loading {
		progressText := "Analyzing files..."
		if m.filesFound > 0 {
			progressText = fmt.Sprintf("Files found: %d, Processed: %d", m.filesFound, m.filesProcessed)
		}
		return progressText + "\n\nPress 'q' to quit"
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

	// Create scroll indicator
	scrollInfo := ""
	if len(m.filteredCounts) > m.maxVisible {
		scrollInfo = fmt.Sprintf(" | View: %d-%d/%d", m.scrollOffset+1, min(m.scrollOffset+m.maxVisible, len(m.filteredCounts)), len(m.filteredCounts))
	}

	// Add label mode indicator
	labelModeInfo := fmt.Sprintf(" | Labels: %s", m.labelMode.String())

	fileStats := fmt.Sprintf("Found: %d | Processed: %d | Files/dirs ignored: %d | Total chars: %d",
		m.result.FilesFound, m.result.FilesFound-m.result.FilesIgnored, m.result.FilesIgnored, m.result.TotalChars)

	timingStats := fmt.Sprintf("Timing: Total %s | Gitignore %s | Traversal %s | Sorting %s",
		m.result.Timing.TotalDuration,
		m.result.Timing.GitignoreDuration,
		m.result.Timing.TraversalDuration,
		m.result.Timing.SortingDuration)

	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(fmt.Sprintf("Directory: %s | Filter: %s%s | Showing: %d/%d chars%s%s", m.directory, m.filterMode.String(), whitespaceStatus, len(m.filteredCounts), len(m.charCounts), scrollInfo, labelModeInfo))

	stats := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Render(fileStats)

	timing := lipgloss.NewStyle().
		Foreground(lipgloss.Color("5")).
		Render(timingStats)

	chart := m.chart.View()

	// bordered window around the chart
	chartWindow := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(1, 2).
		Render(chart)

	controls := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Controls: 'a' all | 'l' letters/numbers | 's' symbols | 'w' toggle whitespace | 't' toggle labels | ←→ scroll | home/end | 'r' refresh | 'q' quit")

	return fmt.Sprintf("%s\n%s\n%s\n%s\n\n%s\n\n%s", title, info, stats, timing, chartWindow, controls)
}
