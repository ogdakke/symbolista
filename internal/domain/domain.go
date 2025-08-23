package domain

import "time"

type CharCount struct {
	Char       string  `json:"char"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type SequenceCount struct {
	Sequence   string  `json:"sequence"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type SequenceCounts []SequenceCount

func (s SequenceCounts) Len() int { return len(s) }
func (s SequenceCounts) Less(i, j int) bool {
	if s[i].Count != s[j].Count {
		return s[i].Count > s[j].Count
	}
	return s[i].Sequence < s[j].Sequence
}
func (s SequenceCounts) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type CharCounts []CharCount

func (c CharCounts) Len() int { return len(c) }
func (c CharCounts) Less(i, j int) bool {
	if c[i].Count != c[j].Count {
		return c[i].Count > c[j].Count
	}
	return c[i].Char < c[j].Char
}
func (c CharCounts) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

type TimingBreakdown struct {
	TotalDuration     time.Duration `json:"total_duration"`
	GitignoreDuration time.Duration `json:"gitignore_duration"`
	TraversalDuration time.Duration `json:"traversal_duration"`
	SortingDuration   time.Duration `json:"sorting_duration"`
	OutputDuration    time.Duration `json:"output_duration"`
}

type AnalysisResult struct {
	CharCounts      CharCounts
	SequenceCounts  SequenceCounts
	FilesFound      int
	FilesIgnored    int
	TotalChars      int
	UniqueChars     int
	UniqueSequences int
	Timing          TimingBreakdown
}

type JSONMetadata struct {
	Directory       string          `json:"directory"`
	FilesFound      int             `json:"files_found"`
	FilesProcessed  int             `json:"files_processed"`
	FilesIgnored    int             `json:"files_ignored"`
	TotalCharacters int             `json:"total_characters"`
	UniqueChars     int             `json:"unique_characters"`
	Timing          TimingBreakdown `json:"timing"`
}

type JSONResult struct {
	Characters CharCounts     `json:"characters"`
	Sequences  SequenceCounts `json:"sequences"`
}

type JSONOutput struct {
	Result   JSONResult    `json:"result"`
	Metadata *JSONMetadata `json:"metadata,omitempty"`
}
