package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/NimbleMarkets/ntcharts/barchart"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ogdakke/symbolista/internal/concurrent"
	"github.com/ogdakke/symbolista/internal/counter"
	"github.com/ogdakke/symbolista/internal/domain"
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

type ViewMode int

const (
	ViewCharacters ViewMode = iota
	ViewSequences
	ViewBigrams
	ViewTrigrams
)

type Model struct {
	directory       string
	showPercentages bool
	workerCount     int
	includeDotfiles bool
	asciiOnly       bool
	topNSeq         int

	charCounts        domain.CharCounts
	sequenceCounts    domain.SequenceCounts
	filteredCounts    domain.CharCounts
	filteredSequences domain.SequenceCounts

	chart             barchart.Model
	ready             bool
	loading           bool
	err               error
	filterMode        FilterMode
	viewMode          ViewMode
	excludeWhitespace bool

	width  int
	height int

	// Horizontal scrolling
	scrollOffset int
	maxVisible   int

	// Label display mode
	labelMode LabelMode

	// File statistics and timing
	result domain.AnalysisResult

	// Progress tracking
	filesFound     int
	filesProcessed int
	progressChan   chan progressMsg
}

type analysisCompleteMsg struct {
	result domain.AnalysisResult
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
	m.filteredSequences = m.filteredSequences[:0]

	// Filter characters
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

	// Filter sequences (apply basic filtering and view mode filtering together)
	for _, seqCount := range m.sequenceCounts {
		includeSequence := true

		// Apply whitespace filtering
		if m.excludeWhitespace {
			allWhitespace := true
			for _, r := range seqCount.Sequence {
				if !isWhitespace(r) {
					allWhitespace = false
					break
				}
			}
			if allWhitespace {
				includeSequence = false
			}
		}

		if includeSequence {
			switch m.viewMode {
			case ViewBigrams:
				if !m.viewMode.FilterBigrams(seqCount) {
					includeSequence = false
				}
			case ViewTrigrams:
				if !m.viewMode.FilterTrigrams(seqCount) {
					includeSequence = false
				}
			}
		}

		if includeSequence {
			switch m.filterMode {
			case FilterAll:
				m.filteredSequences = append(m.filteredSequences, seqCount)
			case FilterLettersNumbers:
				hasLetterNumber := false
				for _, r := range seqCount.Sequence {
					if isLetterOrNumber(r) {
						hasLetterNumber = true
						break
					}
				}
				if hasLetterNumber {
					m.filteredSequences = append(m.filteredSequences, seqCount)
				}
			case FilterSymbols:
				hasSymbol := false
				for _, r := range seqCount.Sequence {
					if isSymbol(r) {
						hasSymbol = true
						break
					}
				}
				if hasSymbol {
					m.filteredSequences = append(m.filteredSequences, seqCount)
				}
			}
		}
	}

	sort.Sort(m.filteredCounts)
	sort.Sort(m.filteredSequences)

	m.scrollOffset = 0
}

func (m ViewMode) FilterBigrams(seq domain.SequenceCount) bool {
	return len(seq.Sequence) == 2
}

func (m ViewMode) FilterTrigrams(seq domain.SequenceCount) bool {
	return len(seq.Sequence) == 3
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

func (v ViewMode) String() string {
	switch v {
	case ViewCharacters:
		return "Characters"
	case ViewSequences:
		return "Sequences"
	case ViewBigrams:
		return "Bigrams"
	case ViewTrigrams:
		return "Trigrams"
	default:
		return "Characters"
	}
}

func NewModel(directory string, showPercentages bool, workerCount int, includeDotfiles bool, asciiOnly bool, topNSeq int) Model {
	return Model{
		directory:         directory,
		showPercentages:   showPercentages,
		workerCount:       workerCount,
		includeDotfiles:   includeDotfiles,
		asciiOnly:         asciiOnly,
		topNSeq:           topNSeq,
		loading:           true,
		filterMode:        FilterAll,
		viewMode:          ViewCharacters,
		excludeWhitespace: true,
	}
}

func NewModelFromJSON(jsonOutput domain.JSONOutput) Model {
	model := Model{
		directory:         "from JSON file",
		showPercentages:   true,
		charCounts:        jsonOutput.Result.Characters,
		sequenceCounts:    jsonOutput.Result.Sequences,
		ready:             true,
		loading:           false,
		filterMode:        FilterAll,
		viewMode:          ViewCharacters,
		excludeWhitespace: true,
	}

	if jsonOutput.Metadata != nil {
		model.directory = jsonOutput.Metadata.Directory
		model.result = domain.AnalysisResult{
			CharCounts:      jsonOutput.Result.Characters,
			SequenceCounts:  jsonOutput.Result.Sequences,
			FilesFound:      jsonOutput.Metadata.FilesFound,
			FilesIgnored:    jsonOutput.Metadata.FilesIgnored,
			TotalChars:      jsonOutput.Metadata.TotalCharacters,
			UniqueChars:     jsonOutput.Metadata.UniqueChars,
			UniqueSequences: len(jsonOutput.Result.Sequences),
			Timing:          jsonOutput.Metadata.Timing,
		}
	} else {
		totalChars := 0
		for _, c := range jsonOutput.Result.Characters {
			totalChars += c.Count
		}
		model.result = domain.AnalysisResult{
			CharCounts:      jsonOutput.Result.Characters,
			SequenceCounts:  jsonOutput.Result.Sequences,
			FilesFound:      0,
			FilesIgnored:    0,
			TotalChars:      totalChars,
			UniqueChars:     len(jsonOutput.Result.Characters),
			UniqueSequences: len(jsonOutput.Result.Sequences),
		}
	}

	model.applyFilter()
	return model
}

func (m Model) Init() tea.Cmd {
	if m.ready {
		return tea.EnterAltScreen
	}
	return tea.Batch(
		startAnalysis(m.directory, m.workerCount, m.includeDotfiles, m.asciiOnly, m.topNSeq),
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

func startAnalysis(directory string, workerCount int, includeDotfiles bool, asciiOnly bool, topNSeq int) tea.Cmd {
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

			// Default sequence config - enabled
			sequenceConfig := concurrent.SequenceConfig{
				Enabled:   true,
				MinLength: 2,
				MaxLength: 3,
				Threshold: 2,
			}

			result, err := counter.AnalyzeSymbols(directory, workerCount, includeDotfiles, asciiOnly, sequenceConfig, progressFunc, topNSeq)

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
		m.sequenceCounts = msg.result.SequenceCounts
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
				return m, startAnalysis(m.directory, m.workerCount, m.includeDotfiles, m.asciiOnly, m.topNSeq)
			}

		case "f":
			if m.ready {
				m.filterMode = (m.filterMode + 1) % 3 // this is the length of options
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
			if m.ready {
				var maxItems int
				switch m.viewMode {
				case ViewCharacters:
					maxItems = len(m.filteredCounts)
				case ViewBigrams, ViewTrigrams, ViewSequences:
					maxItems = len(m.filteredSequences)

				}
				if m.scrollOffset < maxItems-m.maxVisible {
					m.scrollOffset++
					m.updateChart()
				}
			}
		case "home":
			if m.ready {
				m.scrollOffset = 0
				m.updateChart()
			}
		case "end":
			if m.ready {
				var maxItems int
				switch m.viewMode {
				case ViewCharacters:
					maxItems = len(m.filteredCounts)
				case ViewBigrams, ViewTrigrams, ViewSequences:
					maxItems = len(m.filteredSequences)
				}
				if maxItems > m.maxVisible {
					m.scrollOffset = maxItems - m.maxVisible
					m.updateChart()
				}
			}
		case "l":
			if m.ready {
				m.labelMode = (m.labelMode + 1) % 2
				m.updateChart()
			}
		case "m":
			if m.ready {
				m.viewMode = (m.viewMode + 1) % 4
				m.scrollOffset = 0 // Reset scroll when switching views
				m.applyFilter()
				m.updateChart()
			}
		}
	}

	return m, nil
}

func (m *Model) updateChart() {
	if !m.ready {
		return
	}

	// Check if we have data for the current view mode
	var dataLen int
	switch m.viewMode {
	case ViewCharacters:
		dataLen = len(m.filteredCounts)
	case ViewBigrams, ViewTrigrams, ViewSequences:
		dataLen = len(m.filteredSequences)
	}

	if dataLen == 0 {
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
	estimatedLabelWidth := 11
	m.maxVisible = min(chartWidth/estimatedLabelWidth, dataLen, 25)

	// Ensure scroll offset is within bounds
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	maxScrollOffset := max(0, dataLen-m.maxVisible)
	if m.scrollOffset > maxScrollOffset {
		m.scrollOffset = maxScrollOffset
	}

	var barData []barchart.BarData
	colors := []string{"10", "9", "11", "14", "13", "12", "6", "5", "4", "3", "2", "1"}

	// Calculate visible range
	startIndex := m.scrollOffset
	endIndex := min(startIndex+m.maxVisible, dataLen)

	switch m.viewMode {
	case ViewCharacters:
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

	case ViewSequences, ViewBigrams, ViewTrigrams:
		for i := startIndex; i < endIndex; i++ {
			seq := m.filteredSequences[i]
			displaySeq := seq.Sequence

			// Replace whitespace characters with unicode symbols
			displaySeq = strings.ReplaceAll(displaySeq, "\n", "↵")
			displaySeq = strings.ReplaceAll(displaySeq, " ", "⎵")
			displaySeq = strings.ReplaceAll(displaySeq, "\t", "⇥")
			displaySeq = strings.ReplaceAll(displaySeq, "\r", "⏎")

			// Use original index for color consistency across scrolling
			color := colors[i%len(colors)]

			// Format the value based on label mode
			var valueStr string
			switch m.labelMode {
			case LabelCount:
				if seq.Count >= 1000000 {
					valueStr = fmt.Sprintf("%.1fM", float64(seq.Count)/1000000)
				} else if seq.Count >= 1000 {
					valueStr = fmt.Sprintf("%.1fk", float64(seq.Count)/1000)
				} else {
					valueStr = strconv.Itoa(seq.Count)
				}
			case LabelPercentage:
				valueStr = fmt.Sprintf("%.1f%%", seq.Percentage)
			}

			labelWithCount := fmt.Sprintf("%s:%s", displaySeq, valueStr)

			barData = append(barData, barchart.BarData{
				Label: labelWithCount,
				Values: []barchart.BarValue{
					{Name: strconv.Itoa(seq.Count), Value: float64(seq.Count), Style: lipgloss.NewStyle().Foreground(lipgloss.Color(color))},
				},
			})
		}
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

	whitespaceStatus := "| [w]hitespace on"
	if m.excludeWhitespace {
		whitespaceStatus = " | [w]hitespace off"
	}

	// Create scroll indicator and display info based on view mode
	var scrollInfo string
	var displayInfo string

	switch m.viewMode {
	case ViewCharacters:
		if len(m.filteredCounts) > m.maxVisible {
			scrollInfo = fmt.Sprintf(" | View: %d-%d/%d", m.scrollOffset+1, min(m.scrollOffset+m.maxVisible, len(m.filteredCounts)), len(m.filteredCounts))
		}
		displayInfo = fmt.Sprintf("Directory: %s | [m]ode: %s | [f]ilter: %s%s | Showing: %d/%d chars%s | [l]abels: %s",
			m.directory, m.viewMode.String(), m.filterMode.String(), whitespaceStatus, len(m.filteredCounts), len(m.charCounts), scrollInfo, m.labelMode.String())
	default:
		if len(m.filteredSequences) > m.maxVisible {
			scrollInfo = fmt.Sprintf(" | View: %d-%d/%d", m.scrollOffset+1, min(m.scrollOffset+m.maxVisible, len(m.filteredSequences)), len(m.filteredSequences))
		}
		displayInfo = fmt.Sprintf("Directory: %s | [m]ode: %s | [f]ilter: %s%s | Showing: %d/%d sequences%s | [l]abels: %s",
			m.directory, m.viewMode.String(), m.filterMode.String(), whitespaceStatus, len(m.filteredSequences), len(m.sequenceCounts), scrollInfo, m.labelMode.String())

	}

	fileStats := fmt.Sprintf("Found: %d | Processed: %d | Files/dirs ignored: %d | Total chars: %d | Unique sequences: %d",
		m.result.FilesFound, m.result.FilesFound-m.result.FilesIgnored, m.result.FilesIgnored, m.result.TotalChars, m.result.UniqueSequences)

	timingStats := fmt.Sprintf("Timing: Total %s | Gitignore %s | Traversal %s | Sorting %s",
		m.result.Timing.TotalDuration,
		m.result.Timing.GitignoreDuration,
		m.result.Timing.TraversalDuration,
		m.result.Timing.SortingDuration)

	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("5")).
		Render(displayInfo)

	stats := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Render(fileStats)

	timing := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(timingStats)

	chart := m.chart.View()

	// bordered window around the chart
	chartWindow := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(1, 2).
		Render(chart)

	controls := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Render("Controls: 'm' view mode | 'f' char type | 'w' toggle whitespace | 'l' toggle labels | ←→ scroll | home/end | 'r' refresh | 'q' quit")

	return fmt.Sprintf("%s\n%s\n%s\n%s\n\n%s\n\n%s", title, info, stats, timing, chartWindow, controls)
}
